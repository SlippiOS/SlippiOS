// Package netpages serves the LAN sign-in and ISO upload pages.
package netpages

const (
	// MaxUpload is the largest ISO accepted by the upload page.
	MaxUpload = int64(8 * 1024 * 1024 * 1024)

	maxFormBytes = 1 << 20
)
