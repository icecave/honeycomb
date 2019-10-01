package cert

import (
	"crypto/tls"
	"fmt"
	"time"
)

// formatCertificate returns a string describing a certificate, for use in log
// messages.
func formatCertificate(c *tls.Certificate) string {
	d := time.Until(c.Leaf.NotAfter)

	return fmt.Sprintf(
		"'%s', expires at %s (%s), issued by '%s'",
		c.Leaf.Subject.CommonName,
		c.Leaf.NotAfter.Format(time.RFC3339),
		d/time.Second*time.Second,
		c.Leaf.Issuer.CommonName,
	)
}
