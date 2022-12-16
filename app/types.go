package main

type RefUpdatesData struct {
	Name        string `json:"name"`
	OldObjectId string `json:"oldObjectId"`
	NewObjectId string `json:"newObjectId"`
}

type ResourceData struct {
	Repository RepositoryData   `json:"repository"` // Azure
	RefUpdates []RefUpdatesData `json:"refUpdates"` //Azure
}

type RepositoryData struct {
	Name string `json:"name"`
}

type RequestData struct {
	Ref        string         `json:"ref"`        // Github
	Resource   ResourceData   `json:"resource"`   // Azure
	Repository RepositoryData `json:"repository"` // Github
	Before     string         `json:"before"`     // Github
	After      string         `json:"after"`      // Github
	Deleted    bool           `json:"deleted"`    // Github
}
