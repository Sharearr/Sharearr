//go:build linux && container && !dev

package sharearr

var defaultConfigDir = "/config"
var defaultDataDir = "/data"
var defaultLogDir = defaultConfigDir
var defaultDebugEnabled = false
var defaultLogLevel = "info"
