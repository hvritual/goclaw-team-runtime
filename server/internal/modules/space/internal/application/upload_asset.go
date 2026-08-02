package application

// UploadAsset is an application use case. Inject domain and cross-module ports here.
type UploadAsset struct{}

func NewUploadAsset() *UploadAsset { return &UploadAsset{} }
