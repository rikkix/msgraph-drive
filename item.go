package drive

import "time"

type User struct {
	Email       string `json:"email"`
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type Reference struct {
	DriveID   string `json:"driveId"`
	DriveType string `json:"driveType"`
	ID        string `json:"id"`
	Path      string `json:"path"`
}

type Item struct {
	CreatedAt time.Time `json:"createdDateTime"`
	ID        string    `json:"id"`
	LastMod   time.Time `json:"lastModifiedDateTime"`
	Name      string    `json:"name"`
	WebURL    string    `json:"webUrl"`
	Size      int64     `json:"size"`

	CreatedBy struct {
		User `json:"user"`
	} `json:"createdBy"`
	LastModifiedBy struct {
		User `json:"user"`
	} `json:"lastModifiedBy"`

	ParentReference Reference `json:"parentReference"`

	FileSystemInfo struct {
		CreatedDateTime      time.Time `json:"createdDateTime"`
		LastModifiedDateTime time.Time `json:"lastModifiedDateTime"`
	} `json:"fileSystemInfo"`

	Folder *struct {
		ChildCount int `json:"childCount"`
	} `json:"folder,omitempty"`

	DownloadURL string `json:"@microsoft.graph.downloadUrl,omitempty"`

	File *struct {
		MimeType string `json:"mimeType"`
	} `json:"file,omitempty"`

	Image *struct {
		Height int `json:"height"`
		Width  int `json:"width"`
	} `json:"image,omitempty"`
}

func (item *Item) IsFolder() bool {
	return item.Folder != nil
}
