package utils

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/models"
)

var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

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

func PrintProgress(done chan bool, message string, options *models.Options) {
	if options.QuietMode {
		return
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	startTime := time.Now()
	i := 0

	clearLine := "\r                                                                 "

	for {
		select {
		case <-done:
			fmt.Printf("%s\r%s %s ✓\n", clearLine, spinnerChars[0], message)
			return
		case <-ticker.C:
			elapsed := time.Since(startTime).Round(time.Second)
			fmt.Printf("\r%s %s (%s elapsed)   ", spinnerChars[i], message, elapsed)
			i = (i + 1) % len(spinnerChars)
		}
	}
}

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

	var kinds []string
	for kind := range resourcesByKind {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)

	for _, kind := range kinds {
		fmt.Printf("  - %s: %d\n", kind, resourcesByKind[kind])
	}

	if len(namespaces) > 0 {
		var namespaceList []string
		for ns := range namespaces {
			namespaceList = append(namespaceList, ns)
		}
		sort.Strings(namespaceList)
		fmt.Printf("Namespaces: %s\n", strings.Join(namespaceList, ", "))
	}
}

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

func CreateParentDirs(filepath string) error {
	dir := strings.TrimSuffix(filepath, GetFilename(filepath))
	if dir != "" {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

func GetFilename(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}