package version

// Set by linker during build on `make release`.
var Version string
var BuildTime string

// TODO(rh): Print banner at startup with version and build time?
