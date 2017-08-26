package version

import (
	"fmt"
	"runtime"
)

//横幅
var Banner = `
 
                _____.__        
  _____ _____ _/ ____\__| ____  
 /     \\__  \\   __\|  |/  _ \ 
|  Y Y  \/ __ \|  |  |  (  <_> )
|__|_|  (____  /__|  |__|\____/ 
      \/     \/                 
	  
		   Verson %s

`

//版本号
const Binary = "0.2.1"

//输出版本号信息
func Verbose(app string) string {
	return fmt.Sprintf("%s v%s (built w/%s)", app, Binary, runtime.Version())
}

func Show() string {
	return fmt.Sprintf(Banner, Binary)
}
