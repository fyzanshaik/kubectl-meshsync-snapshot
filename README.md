# kubectl MeshSync Snapshot Plugin

A kubectl plugin for capturing Kubernetes cluster state using MeshSync technology and saving it for offline analysis in Meshery.

## Overview

This plugin provides a lightweight alternative to a full Meshery deployment, allowing users to capture a point-in-time snapshot of their Kubernetes cluster's configuration. The snapshot can be
imported into Meshery for visualization and analysis without requiring permanent connectivity between Meshery Server and the cluster.

## Features

-  **Standalone Operation**: Capture cluster state without requiring a full Meshery deployment
-  **Filtering Options**: Focus on specific namespaces, resource types, or use label selectors
-  **Fast Mode**: Quickly capture essential resources with optimized collection
-  **Customizable Output**: Specify output location or use auto-generated timestamped filenames
-  **Read-Only Access**: Operates in read-only mode without modifying cluster state (beyond temporary CRDs)
-  **Clean Interface**: Progress indicators and resource summaries provide clear feedback
-  **Flexible Format**: Structured JSON output compatible with Meshery's import functionality

## Installation

### Prerequisites

-  Kubernetes cluster and `kubectl` configured
-  Access to your cluster with permissions to create/delete CRDs
-  MeshSync binary (automatically downloaded if not found)

### Installing via Krew

```bash
kubectl krew install meshsync-snapshot
```

### Manual Installation

Clone the repository and build from source:

```bash
git clone https://github.com/meshery/kubectl-meshsync_snapshot.git
cd kubectl-meshsync_snapshot
make build
make install
```

## Usage

### Basic Usage

To capture a cluster snapshot with default settings:

```bash
kubectl meshsync-snapshot
```

This will:

1. Start a temporary NATS broker
2. Deploy MeshSync CRDs temporarily
3. Collect resources for 30 seconds
4. Save the snapshot to `meshsync-snapshot.json` in the current directory
5. Clean up all temporary resources

### Command Options

| Option              | Description                                               |
| ------------------- | --------------------------------------------------------- |
| `--output`, `-o`    | Output file path (default: "meshsync-snapshot.json")      |
| `--auto-name`       | Generate filename with timestamp                          |
| `--namespace`, `-n` | Filter resources by namespace                             |
| `--type`, `-t`      | Filter resources by resource type (e.g., Pod, Deployment) |
| `--selector`, `-l`  | Filter resources by label selector (e.g., app=nginx)      |
| `--exclude`         | Comma-separated list of resource types to exclude         |
| `--fast`            | Capture only essential resources with shorter timeout     |
| `--time`            | Collection time in seconds (default: 30)                  |
| `--format`          | Output format: json or yaml (default: "json")             |
| `--quiet`, `-q`     | Minimal output                                            |
| `--verbose`, `-v`   | Detailed output                                           |
| `--preview`         | Show what would be captured without saving                |

### Examples

**Filter by namespace:**

```bash
kubectl meshsync-snapshot --namespace kube-system
```

**Capture only pods:**

```bash
kubectl meshsync-snapshot --type Pod
```

**Use a custom output file:**

```bash
kubectl meshsync-snapshot --output ~/snapshots/my-cluster.json
```

**Generate timestamped filename:**

```bash
kubectl meshsync-snapshot --auto-name
```

**Capture essential resources quickly:**

```bash
kubectl meshsync-snapshot --fast
```

**Filter by label:**

```bash
kubectl meshsync-snapshot --selector app=nginx
```

**Exclude specific resource types:**

```bash
kubectl meshsync-snapshot --exclude "ConfigMap,Secret"
```

**Custom collection time:**

```bash
kubectl meshsync-snapshot --time 60
```

**Preview without capturing:**

```bash
kubectl meshsync-snapshot --preview
```

**Quiet output for scripting:**

```bash
kubectl meshsync-snapshot --quiet
```

## Architecture

The plugin operates through several key components working together:

### Workflow

1. **Setup Phase**:

   -  Start a temporary NATS server for message brokering
   -  Apply MeshSync CRDs and custom resources to the cluster
   -  Start the MeshSync binary as a subprocess

2. **Collection Phase**:

   -  MeshSync discovers resources and publishes them to NATS
   -  Plugin subscribes to NATS topics to collect resources
   -  Resources are filtered based on user options

3. **Output Phase**:

   -  Collected resources are saved to the specified output file
   -  A summary of captured resources is displayed

4. **Cleanup Phase**:
   -  MeshSync process is terminated
   -  CRDs and custom resources are removed
   -  NATS server is shut down

### Component Diagram

```
┌─────────────────┐     ┌────────────────┐    ┌────────────────┐     ┌────────────────┐
│  kubectl plugin │────▶│  NATS Server   │◀───│  MeshSync      │────▶│  Snapshot File │
└─────────────────┘     └────────────────┘    └────────────────┘     └────────────────┘
        │                                             ▲
        │                                             │
        ▼                                             │
┌─────────────────┐                                   │
│  CRD Manager    │───────────────────────────────────┘
└─────────────────┘
```

### MeshSync Integration

The plugin integrates with MeshSync, a component of Meshery responsible for discovering and synchronizing Kubernetes resources. Key aspects of this integration:

1. **Standalone Execution**: Uses MeshSync as a binary rather than a Kubernetes pod
2. **Temporary NATS Broker**: Sets up a temporary message broker for MeshSync to publish resources
3. **Custom Resource Configuration**: Creates appropriate custom resources for MeshSync to operate
4. **Resource Processing**: Subscribes to published resources and processes them for output

## Technical Details

### MeshSync Operational Flow

MeshSync discovers resources through the following process:

1. **Dynamic Informers**: Utilizes Kubernetes dynamic informers to watch all resource types
2. **Pipeline Processing**: Processes discovered resources through a pipeline
3. **Publishing**: Publishes resources to NATS topics
4. **Resource Model**: Transforms Kubernetes objects into a standardized model

### Plugin Implementation

The implementation addresses several technical challenges:

1. **Process Management**: Carefully manages subprocess execution and cleanup
2. **NATS Configuration**: Sets up a properly configured NATS server for MeshSync to use
3. **CRD Requirements**: Applies necessary CRDs and custom resources for MeshSync operation
4. **Output Control**: Handles verbose logs and error messages for a clean user experience
5. **Resource Collection**: Efficiently collects and processes published resources

### Error Handling

The plugin includes several error handling mechanisms:

1. **Graceful Degradation**: Falls back to different collection strategies when needed
2. **Comprehensive Cleanup**: Ensures all temporary resources are cleaned up even on failure
3. **Clear Feedback**: Provides meaningful error messages and warnings

## Potential Improvements

While the current implementation is functional, several improvements could be made:

1. **Direct MeshSync Integration**: A fork of MeshSync specifically for snapshot functionality could eliminate dependency on NATS
2. **Resource Discovery Optimization**: Enhanced filtering options could reduce unnecessary resource discovery
3. **YAML Output Format**: Supporting YAML output format would provide more flexibility
4. **Collection Progress**: More granular progress reporting based on discovered resource types
5. **Cluster Adaptation**: Automatic adjustment of collection strategy based on cluster size

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Acknowledgments

-  The [Meshery](https://meshery.io/) project for creating MeshSync
-  The [Krew](https://krew.sigs.k8s.io/) project for the kubectl plugin ecosystem
