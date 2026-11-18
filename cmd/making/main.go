package main

import (
 "flag"
 "fmt"
 "github.com/Cheems-gta/Making/core"
 "os"
 "os/signal"
 "syscall"
)

func main() {
 cfg:=flag.String("config","config.json","JSON configuration path")
 version:=flag.Bool("version",false,"print version")
 flag.Parse()
 if *version {fmt.Println("Making Standalone 0.1.0 (Windows/Linux foundation)");return}
 c,e:=core.Load(*cfg)
 if e!=nil {fmt.Fprintln(os.Stderr,"configuration error:",e);os.Exit(2)}
 s,e:=core.New(c)
 if e!=nil {fmt.Fprintln(os.Stderr,e);os.Exit(2)}
 if e=s.Start();e!=nil {fmt.Fprintln(os.Stderr,"startup failed:",e);os.Exit(1)}
 fmt.Println("Making ready (local proxy active; remote endpoint not yet verified)")
 for _,a:=range s.Addrs() {fmt.Println("listening:",a)}
 ch:=make(chan os.Signal,1)
 signal.Notify(ch,os.Interrupt,syscall.SIGTERM)
 <-ch
 s.Stop()
}
