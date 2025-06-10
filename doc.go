/*
Package atmosphere provides a CLI tool for managing cloud infrastructure deployments.

TODO:
  - add config file to allow configuring services (with optional values overrides)
  - add ci + role and start using in github
  - nice to have: add `atmosphere logs` command to see logs of specific services

Usage:

	atmosphere [command]

Available Commands:

	deploy      Deploy infrastructure components
	ovn-nbctl   Execute ovn-nbctl commands on the OVN northbound database
	ovn-sbctl   Execute ovn-sbctl commands on the OVN southbound database

Examples:

	# Deploy all components
	atmosphere deploy

	# List OVN logical routers
	atmosphere ovn-nbctl list Logical_Router

	# List OVN chassis
	atmosphere ovn-sbctl list Chassis

The ovn-nbctl and ovn-sbctl commands will automatically connect to the appropriate
OVN database pods in the openstack namespace and execute the requested commands.
*/
package atmosphere
