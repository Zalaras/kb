package kb

import "github.com/Zalaras/kb/internal/finding"

// Finding is one problem a load or check reports, path-qualified and sorted for output; the
// leaf packages report the same type, so their findings join the index's without copying.
type Finding = finding.Finding
