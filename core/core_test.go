package core

import (
 "bufio"
 "encoding/binary"
 "io"
 "net"
 "testing"
 "time"
)
func readExactly(t *testing.T,c net.Conn,n int)[]byte{
 t.Helper();b:=make([]byte,n)
 if _,e:=io.ReadFull(c,b);e!=nil{t.Fatal(e)}
 return b
}
func TestValidation(t *testing.T){
 c:=Config{ListenSOCKS:"0.0.0.0:1080",Username:"user",Password:"long-password-here",Upstream:"socks5://127.0.0.1:9999"}
 if c.Validate()==nil{t.Fatal("wildcard listener accepted")}
 c.ListenSOCKS="127.0.0.1:1080"
 if e:=c.Validate();e!=nil{t.Fatal(e)}
 c.Upstream="direct://example.com:80"
 if c.Validate()==nil{t.Fatal("DIRECT fallback accepted")}
}
func TestSOCKSOverUpstreamHTTP(t *testing.T){
 echo,e:=net.Listen("tcp","127.0.0.1:0");if e!=nil{t.Fatal(e)};defer echo.Close()
 go func(){for{c,e:=echo.Accept();if e!=nil{return};go func(){defer c.Close();io.Copy(c,c)}()}}()
 upstream,e:=net.Listen("tcp","127.0.0.1:0");if e!=nil{t.Fatal(e)};defer upstream.Close()
 go func(){for{
  c,e:=upstream.Accept();if e!=nil{return}
  go func(){
   defer c.Close();r:=bufio.NewReader(c)
   line,e:=r.ReadString('\n');if e!=nil||len(line)<8||line[:8]!="CONNECT "{return}
   for{h,e:=r.ReadString('\n');if e!=nil{return};if h=="\r\n"{break}}
   dest,e:=net.Dial("tcp",echo.Addr().String());if e!=nil{return};defer dest.Close()
   c.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
   go io.Copy(dest,c)
   io.Copy(c,dest)
  }()
 }}()
 tmp,e:=net.Listen("tcp","127.0.0.1:0");if e!=nil{t.Fatal(e)}
 addr:=tmp.Addr().String();tmp.Close()
 cfg:=Config{ListenSOCKS:addr,Username:"making",Password:"long-test-password",Upstream:"http://"+upstream.Addr().String()}
 s,e:=New(cfg);if e!=nil{t.Fatal(e)}
 if e=s.Start();e!=nil{t.Fatal(e)}
 c,e:=net.DialTimeout("tcp",addr,time.Second);if e!=nil{t.Fatal(e)}
 defer func(){c.Close();s.Stop()}()
 c.SetDeadline(time.Now().Add(5*time.Second))
 c.Write([]byte{5,1,2});if b:=readExactly(t,c,2);b[1]!=2{t.Fatal("method",b)}
 user,pw:="making","long-test-password"
 pkt:=append([]byte{1,byte(len(user))},[]byte(user)...)
 pkt=append(pkt,byte(len(pw)));pkt=append(pkt,[]byte(pw)...)
 c.Write(pkt);if b:=readExactly(t,c,2);b[1]!=0{t.Fatal("auth rejected")}
 pkt=[]byte{5,1,0,1,127,0,0,1}
 _,p,e:=net.SplitHostPort(echo.Addr().String());if e!=nil{t.Fatal(e)}
 port:=0;for _,ch:=range p{port=port*10+int(ch-'0')}
 pkt=binary.BigEndian.AppendUint16(pkt,uint16(port));c.Write(pkt)
 if b:=readExactly(t,c,10);b[1]!=0{t.Fatal("connect",b)}
 c.Write([]byte("making-test"))
 if string(readExactly(t,c,len("making-test")))!="making-test"{t.Fatal("relay mismatch")}
}
