package helm

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"helm.sh/helm/v3/pkg/release"
)

// ConfigHashKey is the annotation key used to store the configuration hash
const ConfigHashKey = "atmosphere.dev/config-hash"

// calculateConfigHash returns a hash of the release values, excluding metadata
func calculateConfigHash(values map[string]interface{}) string {
	// Create a copy of values, excluding metadata that shouldn't affect the hash
	cleanValues := make(map[string]interface{})
	for k, v := range values {
		if k != "annotations" && k != "labels" {
			cleanValues[k] = v
		}
	}

	// Marshal to JSON for consistent ordering
	valuesBytes, err := json.Marshal(cleanValues)
	if err != nil {
		log.Error("Failed to marshal values for hashing", "error", err)
		return fmt.Sprintf("error-%d", time.Now().Unix())
	}

	// Calculate SHA256 hash
	hasher := sha256.New()
	hasher.Write(valuesBytes)
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// AddConfigHash adds a hash of the configuration values to the release annotations
func AddConfigHash(values map[string]interface{}) map[string]interface{} {
	// Make a copy of the values to avoid modifying the original
	valuesCopy := make(map[string]interface{})
	for k, v := range values {
		valuesCopy[k] = v
	}

	// Calculate hash of the core configuration (excluding metadata)
	configHash := calculateConfigHash(valuesCopy)

	// Ensure annotations exist
	if valuesCopy["annotations"] == nil {
		valuesCopy["annotations"] = make(map[string]interface{})
	}

	// Add hash to annotations
	annotations := valuesCopy["annotations"].(map[string]interface{})
	annotations[ConfigHashKey] = configHash[:8] // Use first 8 chars for brevity
	annotations["atmosphere.dev/managed-by"] = "atmosphere-cli"

	log.Debug("Added config hash to release values", "hash", configHash[:8])

	return valuesCopy
}

// NeedsUpdate checks if a release needs to be updated based on config hash
func NeedsUpdate(newValues map[string]interface{}, existingRelease *release.Release) bool {
	// Check if newValues has our hash annotation
	newAnnotations, hasNewAnnotations := newValues["annotations"].(map[string]interface{})
	if !hasNewAnnotations || newAnnotations[ConfigHashKey] == nil {
		// No hash in new values, can't use hash-based comparison
		return true
	}

	newHash := newAnnotations[ConfigHashKey]

	// Check if existing release has a hash
	var existingHash interface{}
	if existingRelease != nil && existingRelease.Config != nil {
		if existingAnnotations, ok := existingRelease.Config["annotations"].(map[string]interface{}); ok {
			existingHash = existingAnnotations[ConfigHashKey]
		}
	}

	// If hashes differ or no previous hash, update is needed
	if existingHash != newHash {
		log.Debug("Config hash changed, release needs update",
			"oldHash", existingHash,
			"newHash", newHash)
		return true
	}

	log.Debug("Config hash unchanged, no update needed", "hash", newHash)
	return false
}
