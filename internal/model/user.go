package model

type UserProfile struct {
    ID            string `json:"id,omitempty"`
    Name          string `json:"name,omitempty"`
    Email         string `json:"email,omitempty"`
    Mobile        string `json:"mobile,omitempty"`
    Status        string `json:"status,omitempty"`
    LastUpdatedAt string `json:"last_updated_at,omitempty"`
	 SyncStatus    string `json:"-"` 
    SyncMessage   string `json:"-"`
}
