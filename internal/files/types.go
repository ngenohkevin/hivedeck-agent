package files

import "time"

// FileInfo represents a file or directory
type FileInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	Mode        string    `json:"mode"`
	ModTime     time.Time `json:"mod_time"`
	IsDir       bool      `json:"is_dir"`
	IsSymlink   bool      `json:"is_symlink"`
	LinkTarget  string    `json:"link_target,omitempty"`
	Owner       string    `json:"owner"`
	Group       string    `json:"group"`
	Permissions string    `json:"permissions"`
}

// DirectoryListing represents a directory and its contents
type DirectoryListing struct {
	Path    string     `json:"path"`
	Files   []FileInfo `json:"files"`
	Total   int        `json:"total"`
	CanRead bool       `json:"can_read"`
}

// FileContent represents the content of a file
type FileContent struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Size     int64  `json:"size"`
	Encoding string `json:"encoding"` // "utf-8" or "base64"
	IsBinary bool   `json:"is_binary"`
	Truncated bool  `json:"truncated"`
}

// DiskUsageInfo represents disk usage for a path
type DiskUsageInfo struct {
	Path       string `json:"path"`
	TotalSize  int64  `json:"total_size"`
	FileCount  int    `json:"file_count"`
	DirCount   int    `json:"dir_count"`
	LargestFiles []FileInfo `json:"largest_files,omitempty"`
}
