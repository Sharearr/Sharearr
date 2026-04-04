//go:build linux && !dev && !container

package sharearr

var defaultConfigDir = "/etc/sharearr"
var defaultDataDir = "/var/lib/sharearr"
var defaultLogDir = "/var/log/sharearr"
var defaultDebugEnabled = false
var defaultLogLevel = "info"
