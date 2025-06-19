package manifests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/charmbracelet/log"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	krusty "sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// NewClient creates a new client for applying Kubernetes manifests
func NewClient(getter genericclioptions.RESTClientGetter, namespace string) (*Client, error) {
	config, err := getter.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get Kubernetes config: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	discovery, err := getter.ToDiscoveryClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	apiGroups, err := restmapper.GetAPIGroupResources(discovery)
	if err != nil {
		return nil, fmt.Errorf("failed to get API group resources: %w", err)
	}

	return &Client{
		dynamicClient: dynamicClient,
		restMapper:    restmapper.NewDiscoveryRESTMapper(apiGroups),
		namespace:     namespace,
	}, nil
}

// getManifestContent retrieves manifest content from various sources
func (c *Client) getManifestContent(source ManifestSource) ([]byte, error) {
	// Handle file paths
	if source.Path != "" {
		matches, err := filepath.Glob(source.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve glob pattern: %w", err)
		}

		if len(matches) == 0 {
			return nil, fmt.Errorf("no files match pattern: %s", source.Path)
		}

		var allContent bytes.Buffer
		for _, match := range matches {
			content, err := os.ReadFile(match)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", match, err)
			}
			allContent.Write(content)
			allContent.WriteString("\n---\n")
		}
		return allContent.Bytes(), nil
	}

	// Handle URL
	if source.URL != "" {
		resp, err := http.Get(source.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch manifest from URL %s: %w", source.URL, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("received non-OK status code %d from URL %s", resp.StatusCode, source.URL)
		}

		content, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body from URL %s: %w", source.URL, err)
		}
		return content, nil
	}

	// Handle raw content
	if source.Content != "" {
		return []byte(source.Content), nil
	}

	// Handle inline Kustomize content
	if source.KustomizeContent != "" {
		// Create a temporary directory for the kustomization
		tmpDir, err := os.MkdirTemp("", "kustomize-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temporary directory for kustomize: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		// Write the kustomization.yaml file
		kustomizationPath := filepath.Join(tmpDir, "kustomization.yaml")
		if err := os.WriteFile(kustomizationPath, []byte(source.KustomizeContent), 0644); err != nil {
			return nil, fmt.Errorf("failed to write kustomization.yaml: %w", err)
		}

		log.Debug("Created temporary kustomization", "path", kustomizationPath)

		// Use the native kustomize library
		opts := krusty.MakeDefaultOptions()
		kustomizer := krusty.MakeKustomizer(opts)
		fs := filesys.MakeFsOnDisk()

		// Build the resources using the kustomize library
		resMap, err := kustomizer.Run(fs, tmpDir)
		if err != nil {
			return nil, fmt.Errorf("failed to build kustomize resources from inline content: %w", err)
		}

		// Convert the resource map to YAML
		yaml, err := resMap.AsYaml()
		if err != nil {
			return nil, fmt.Errorf("failed to convert inline kustomize resources to YAML: %w", err)
		}

		return yaml, nil
	}

	// Handle Kustomize paths
	if source.KustomizePath != "" {
		// Use the native kustomize library instead of the CLI tool
		opts := krusty.MakeDefaultOptions()
		kustomizer := krusty.MakeKustomizer(opts)

		// Get the directory and create a filesystem
		fs := filesys.MakeFsOnDisk()

		// Resolve the absolute path if it's relative
		absPath := source.KustomizePath
		if !filepath.IsAbs(absPath) {
			var err error
			absPath, err = filepath.Abs(absPath)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve absolute path for kustomize: %w", err)
			}
		}

		log.Debug("Building kustomize resources", "path", absPath)

		// Build the resources using the kustomize library
		resMap, err := kustomizer.Run(fs, absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build kustomize resources: %w", err)
		}

		// Convert the resource map to YAML
		yaml, err := resMap.AsYaml()
		if err != nil {
			return nil, fmt.Errorf("failed to convert kustomize resources to YAML: %w", err)
		}

		return yaml, nil
	}

	// Handle templates
	if source.Template != "" {
		tmpl, err := template.New("manifest").Parse(source.Template)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template: %w", err)
		}

		var rendered bytes.Buffer
		if err := tmpl.Execute(&rendered, source.TemplateValues); err != nil {
			return nil, fmt.Errorf("failed to render template: %w", err)
		}

		return rendered.Bytes(), nil
	}

	return nil, fmt.Errorf("no valid manifest source specified")
}

// ApplyManifests applies Kubernetes manifests from the specified sources
func (c *Client) ApplyManifests(ctx context.Context, config *ManifestConfig) error {
	// Set a default namespace if none is specified
	namespace := "default"
	if config.Namespace != "" {
		namespace = config.Namespace
	} else if c.namespace != "" {
		namespace = c.namespace
	}

	log.Info("Applying manifests", "namespace", namespace)

	// Note: Namespace creation is now handled separately in EnsureNamespaceExists
	// which is called before ApplyManifests in the component task

	// Process each manifest source
	for i, source := range config.Sources {
		content, err := c.getManifestContent(source)
		if err != nil {
			return fmt.Errorf("failed to get manifest content from source #%d: %w", i, err)
		}

		if err := c.applyManifestContent(ctx, content, namespace); err != nil {
			return fmt.Errorf("failed to apply manifest from source #%d: %w", i, err)
		}
	}

	return nil
}

// EnsureNamespaceExists ensures that the specified namespace exists
func (c *Client) EnsureNamespaceExists(ctx context.Context, namespace string) error {
	return c.createNamespaceIfNotExists(ctx, namespace)
}

// createNamespaceIfNotExists creates a namespace if it doesn't already exist
func (c *Client) createNamespaceIfNotExists(ctx context.Context, namespace string) error {
	// Create a direct API call to create the namespace
	nsObj := &unstructured.Unstructured{}
	nsObj.SetAPIVersion("v1")
	nsObj.SetKind("Namespace")
	nsObj.SetName(namespace)

	// Get the API resource for namespace
	mapping, err := c.restMapper.RESTMapping(schema.GroupKind{Group: "", Kind: "Namespace"}, "v1")
	if err != nil {
		return fmt.Errorf("failed to get REST mapping for Namespace: %w", err)
	}

	// Get the dynamic resource interface for namespaces (which are cluster-scoped)
	dr := c.dynamicClient.Resource(mapping.Resource)

	// Check if namespace exists
	_, err = dr.Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		// Namespace already exists
		log.Debug("Namespace already exists", "name", namespace)
		return nil
	}

	log.Info("Creating namespace", "name", namespace)

	// Create the namespace
	result, err := dr.Create(ctx, nsObj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
	}

	log.Info("Created namespace", "name", result.GetName())

	// Give the Kubernetes API server a moment to process the namespace creation
	// This helps prevent race conditions where resources are created before the namespace is fully ready
	time.Sleep(2 * time.Second)

	// Verify namespace exists after creation
	_, verifyErr := dr.Get(ctx, namespace, metav1.GetOptions{})
	if verifyErr != nil {
		return fmt.Errorf("namespace %s was not created properly: %w", namespace, verifyErr)
	}

	return nil
	return nil
}

// applyManifestContent applies the given manifest content
func (c *Client) applyManifestContent(ctx context.Context, content []byte, defaultNamespace string) error {
	decoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	// Split the YAML documents
	parts := bytes.Split(content, []byte("---\n"))

	for _, part := range parts {
		// Skip empty parts
		if len(bytes.TrimSpace(part)) == 0 {
			continue
		}

		obj := &unstructured.Unstructured{}
		_, gvk, err := decoder.Decode(part, nil, obj)
		if err != nil {
			log.Warn("Failed to decode manifest", "error", err)
			continue
		}

		// Set namespace if not specified and not a cluster-scoped resource
		if defaultNamespace != "" {
			if obj.GetNamespace() == "" {
				// Check if this resource type supports namespaces
				mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
				if err != nil {
					return fmt.Errorf("failed to get REST mapping: %w", err)
				}

				if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
					obj.SetNamespace(defaultNamespace)
				}
			}
		}

		// Get the API resource for this GVK
		mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return fmt.Errorf("failed to get REST mapping for %s: %w", gvk.String(), err)
		}

		// Get the dynamic resource interface
		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			dr = c.dynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
		} else {
			dr = c.dynamicClient.Resource(mapping.Resource)
		}

		// Try to get the existing resource
		name := obj.GetName()
		existingObj, err := dr.Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			// Resource exists, update it
			// Set the resourceVersion to ensure we're updating the latest version
			obj.SetResourceVersion(existingObj.GetResourceVersion())
			result, err := dr.Update(ctx, obj, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update resource %s/%s: %w", gvk.Kind, name, err)
			}
			log.Info("Updated resource",
				"kind", gvk.Kind,
				"name", result.GetName(),
				"namespace", result.GetNamespace(),
			)
		} else {
			// Resource doesn't exist, create it
			result, err := dr.Create(ctx, obj, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create resource %s/%s: %w", gvk.Kind, name, err)
			}
			log.Info("Created resource",
				"kind", gvk.Kind,
				"name", result.GetName(),
				"namespace", result.GetNamespace(),
			)
		}
	}

	return nil
}
