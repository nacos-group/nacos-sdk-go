package cache

type ConfigCachedFileType string

const (
	ConfigContent          ConfigCachedFileType = "Config Content"
	ConfigEncryptedDataKey ConfigCachedFileType = "Config Encrypted Data Key"

	ENCRYPTED_DATA_KEY_FILE_NAME = "encrypted-data-key"
	FAILOVER_FILE_SUFFIX         = "_failover"
)
