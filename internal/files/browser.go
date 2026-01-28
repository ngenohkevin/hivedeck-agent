package files

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf8"
)

const (
	// MaxFileSize is the maximum file size that can be read (1MB)
	MaxFileSize = 1 * 1024 * 1024
	// MaxDirEntries is the maximum number of directory entries to return
	MaxDirEntries = 1000
)

// Browser handles file system operations (read-only)
type Browser struct {
	allowedPaths []string
	allowAll     bool
}

// NewBrowser creates a new file browser
func NewBrowser(allowedPaths []string) *Browser {
	// Check for wildcard "*" which means allow all paths
	allowAll := false
	for _, p := range allowedPaths {
		if p == "*" {
			allowAll = true
			break
		}
	}

	if !allowAll && len(allowedPaths) == 0 {
		// Default allowed paths
		allowedPaths = []string{
			"/var/log",
			"/etc",
			"/home",
			"/opt",
			"/tmp",
		}
	}
	return &Browser{
		allowedPaths: allowedPaths,
		allowAll:     allowAll,
	}
}

// GetAllowedPaths returns the list of allowed paths for the UI
func (b *Browser) GetAllowedPaths() []string {
	if b.allowAll {
		return []string{"/"}
	}
	return b.allowedPaths
}

// IsPathAllowed checks if a path is within allowed directories
func (b *Browser) IsPathAllowed(path string) bool {
	if b.allowAll {
		return true
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Clean the path to prevent directory traversal
	absPath = filepath.Clean(absPath)

	for _, allowed := range b.allowedPaths {
		allowedAbs, err := filepath.Abs(allowed)
		if err != nil {
			continue
		}
		allowedAbs = filepath.Clean(allowedAbs)

		if strings.HasPrefix(absPath, allowedAbs) {
			return true
		}
	}

	return false
}

// ListDirectory returns the contents of a directory
func (b *Browser) ListDirectory(path string) (*DirectoryListing, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	if !b.IsPathAllowed(absPath) {
		return nil, fmt.Errorf("access denied: path not in allowed list")
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return &DirectoryListing{
			Path:    absPath,
			Files:   []FileInfo{},
			Total:   0,
			CanRead: false,
		}, nil
	}

	var files []FileInfo
	for i, entry := range entries {
		if i >= MaxDirEntries {
			break
		}

		fileInfo, err := b.getFileInfo(filepath.Join(absPath, entry.Name()))
		if err != nil {
			continue
		}
		files = append(files, *fileInfo)
	}

	// Sort: directories first, then by name
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Name < files[j].Name
	})

	return &DirectoryListing{
		Path:    absPath,
		Files:   files,
		Total:   len(files),
		CanRead: true,
	}, nil
}

// ReadFile returns the content of a file
func (b *Browser) ReadFile(path string) (*FileContent, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	if !b.IsPathAllowed(absPath) {
		return nil, fmt.Errorf("access denied: path not in allowed list")
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory")
	}

	// Check file size
	truncated := false
	readSize := info.Size()
	if readSize > MaxFileSize {
		readSize = MaxFileSize
		truncated = true
	}

	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	content := make([]byte, readSize)
	n, err := io.ReadFull(file, content)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	content = content[:n]

	// Check if content is valid UTF-8
	isBinary := !utf8.Valid(content)
	encoding := "utf-8"
	if isBinary {
		encoding = "binary"
	}

	return &FileContent{
		Path:      absPath,
		Content:   string(content),
		Size:      info.Size(),
		Encoding:  encoding,
		IsBinary:  isBinary,
		Truncated: truncated,
	}, nil
}

// GetDiskUsage returns disk usage information for a path
func (b *Browser) GetDiskUsage(path string) (*DiskUsageInfo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	if !b.IsPathAllowed(absPath) {
		return nil, fmt.Errorf("access denied: path not in allowed list")
	}

	var totalSize int64
	var fileCount, dirCount int
	var largestFiles []FileInfo

	err = filepath.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if d.IsDir() {
			dirCount++
		} else {
			fileCount++
			info, err := d.Info()
			if err == nil {
				totalSize += info.Size()

				// Track largest files
				fileInfo := FileInfo{
					Name: d.Name(),
					Path: path,
					Size: info.Size(),
				}
				largestFiles = append(largestFiles, fileInfo)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Sort and keep top 10 largest files
	sort.Slice(largestFiles, func(i, j int) bool {
		return largestFiles[i].Size > largestFiles[j].Size
	})
	if len(largestFiles) > 10 {
		largestFiles = largestFiles[:10]
	}

	return &DiskUsageInfo{
		Path:         absPath,
		TotalSize:    totalSize,
		FileCount:    fileCount,
		DirCount:     dirCount,
		LargestFiles: largestFiles,
	}, nil
}

func (b *Browser) getFileInfo(path string) (*FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}

	fileInfo := &FileInfo{
		Name:        info.Name(),
		Path:        path,
		Size:        info.Size(),
		Mode:        info.Mode().String(),
		ModTime:     info.ModTime(),
		IsDir:       info.IsDir(),
		IsSymlink:   info.Mode()&os.ModeSymlink != 0,
		Permissions: info.Mode().Perm().String(),
	}

	// Get symlink target
	if fileInfo.IsSymlink {
		target, err := os.Readlink(path)
		if err == nil {
			fileInfo.LinkTarget = target
		}
	}

	// Get owner and group
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		if u, err := user.LookupId(strconv.Itoa(int(stat.Uid))); err == nil {
			fileInfo.Owner = u.Username
		} else {
			fileInfo.Owner = strconv.Itoa(int(stat.Uid))
		}
		if g, err := user.LookupGroupId(strconv.Itoa(int(stat.Gid))); err == nil {
			fileInfo.Group = g.Name
		} else {
			fileInfo.Group = strconv.Itoa(int(stat.Gid))
		}
	}

	return fileInfo, nil
}
