package handler

import "strings"

// joinResourceURL builds the public URL for an object stored in S3-compatible
// storage. The configured domain is already the root of the configured bucket;
// bucket names belong only in S3 API requests, never in the public URL.
func joinResourceURL(domain, objectKey string) string {
	return strings.TrimRight(domain, "/") + "/" + strings.TrimLeft(objectKey, "/")
}
