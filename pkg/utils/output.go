package utils

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/models"
)

// Spinner characters for animation
var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// FormatSize formats bytes into a human-readable size
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// PrintProgress displays a spinner with a message
func PrintProgress(done chan bool, message string, options *models.Options) {
	if options.QuietMode {
		return
	}
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	startTime := time.Now()
	i := 0
	
	for {
		select {
		case <-done:
			fmt.Printf("\r%s %s ✓                                 \n", spinnerChars[0], message)
			return
		case <-ticker.C:
			elapsed := time.Since(startTime).Round(time.Second)
			fmt.Printf("\r%s %s (%s elapsed)   ", spinnerChars[i], message, elapsed)
			i = (i + 1) % len(spinnerChars)
		}
	}
}

// PrintResourceSummary displays a summary of captured resources
func PrintResourceSummary(resources []*models.KubernetesResource, options *models.Options) {
	if options.QuietMode {
		return
	}
	
	resourcesByKind := make(map[string]int)
	namespaces := make(map[string]bool)
	
	for _, res := range resources {
		resourcesByKind[res.Kind]++
		if res.KubernetesResourceMeta != nil && res.KubernetesResourceMeta.Namespace != "" {
			namespaces[res.KubernetesResourceMeta.Namespace] = true
		}
	}
	
	fmt.Printf("Resource Summary:\n")
	
	// Get a sorted list of resource kinds for consistent output
	var kinds []string
	for kind := range resourcesByKind {
		kinds = append(kinds, kind)
	}
	
	// Print resource counts by kind
	for _, kind := range kinds {
		fmt.Printf("  - %s: %d\n", kind, resourcesByKind[kind])
	}
	
	// Print namespace summary if more than one
	if len(namespaces) > 0 {
		var namespaceList []string
		for ns := range namespaces {
			namespaceList = append(namespaceList, ns)
		}
		fmt.Printf("Namespaces: %s\n", strings.Join(namespaceList, ", "))
	}
}

// GenerateTimestampedFilename generates a filename with a timestamp
func GenerateTimestampedFilename(baseFilename string) string {
	timestamp := time.Now().Format("20060102-150405")
	
	ext := ".json"
	basename := baseFilename
	
	if strings.HasSuffix(baseFilename, ".json") {
		basename = baseFilename[:len(baseFilename)-5]
	} else if strings.HasSuffix(baseFilename, ".yaml") || strings.HasSuffix(baseFilename, ".yml") {
		ext = baseFilename[len(baseFilename)-5:]
		basename = baseFilename[:len(baseFilename)-5]
	}
	
	return fmt.Sprintf("%s-%s%s", basename, timestamp, ext)
}

// CreateParentDirs ensures parent directories exist for a file path
func CreateParentDirs(filepath string) error {
	dir := strings.TrimSuffix(filepath, GetFilename(filepath))
	if dir != "" {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// GetFilename returns just the filename part of a path
func GetFilename(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}