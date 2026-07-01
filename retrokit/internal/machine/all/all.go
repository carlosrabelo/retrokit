// Package all imports every machine implementation for its registration side
// effects. Importing this package (blank) ensures all built-in machines are
// available through the machine registry.
//
// Add new platform packages here as they are implemented.
package all

import (
	_ "github.com/carlosrabelo/retrokit/retrokit/internal/machine/msx"
	_ "github.com/carlosrabelo/retrokit/retrokit/internal/machine/trs80"
)
