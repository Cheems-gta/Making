package core

import (
 "bufio"
 "context"
 "encoding/base64"
 "encoding/binary"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net"
 "net/http"
 "net/url"
 "os"
 "strconv"
 "strings"
 "sync"
 "time"
)

// Config requires an explicit upstream. No direct fallback is permitted.
type Config struct {
 ListenSOCKS string `json:"listen_socks"`
 ListenHTTP string `json:"listen_http"`
 Username string `json:"username"`
 Password string `json:"password"`
 Upstream string `json:"upstream"`
}

func Load(path string)(Config,error){
 f,e:=os.Open(path);if e!=nil{return Config{},e};defer f.Close()
 var c Config
 d:=json.NewDecoder(io.LimitReader(f,1<<20));d.DisallowUnknownFields()
 if e=d.Decode(&c);e!=nil{return c,e}
 return c,c.Validate()
}
func (c Config) Validate()error{
 if c.ListenSOCKS==""&&c.ListenHTTP==""{return errors.New("at least one local listener required")}
 if c.Username==""||len(c.Password)<12{return errors.New("local proxy auth required (password at least 12 characters)")}
 u,e:=url.Parse(c.Upstream);if e!=nil{return e}
 if u.Scheme!="socks5"&&u.Scheme!="http"{return errors.New("upstream must be socks5:// or http://")}
 if u.Hostname()==""||u.Port()==""{return errors.New("upstream host and port required")}
 for _,addr:=range []string{c.ListenSOCKS,c.ListenHTTP}{
  if addr==""{continue}
  h,p,e:=net.SplitHostPort(addr);if e!=nil{return e}
  if h!="127.0.0.1"&&h!="::1"&&h!="localhost"{return errors.New("listeners must bind loopback")}
  n,e:=strconv.Atoi(p);if e!=nil||n<1||n>65535{return errors.New("invalid listener port")}
 }
 if c.ListenSOCKS!=""&&c.ListenSOCKS==c.ListenHTTP{return errors.New("listeners must differ")}
 return nil
}

type Server struct {
 cfg Config
 listeners []net.Listener
 wg sync.WaitGroup
 mu sync.Mutex
 closed bool
}
func New(c Config)(*Server,error){if e:=c.Validate();e!=nil{return nil,e};return &Server{cfg:c},nil}
func (s *Server) Start()error{
 s.mu.Lock();defer s.mu.Unlock()
 if s.closed{return errors.New("server stopped")}
 if len(s.listeners)!=0{return errors.New("already started")}
 for _,item:=range []struct{addr string;isHTTP bool}{{s.cfg.ListenSOCKS,false},{s.cfg.ListenHTTP,true}}{
  if item.addr==""{continue}
  ln,e:=net.Listen("tcp",item.addr)
  if e!=nil{for _,l:=range s.listeners{l.Close()};s.listeners=nil;return e}
  s.listeners=append(s.listeners,ln)
  s.wg.Add(1)
  go func(l net.Listener,isHTTP bool){
   defer s.wg.Done()
   for {
    c,e:=l.Accept();if e!=nil{return}
    s.wg.Add(1)
    go func(conn net.Conn){
     defer s.wg.Done();defer conn.Close()
     _=conn.SetDeadline(time.Now().Add(2*time.Minute))
     if isHTTP{s.handleHTTP(conn)}else{s.handleSOCKS(conn)}
    }(c)
   }
  }(ln,item.isHTTP)
 }
 return nil
}
func (s *Server) Stop(){
 s.mu.Lock()
 if s.closed{s.mu.Unlock();return}
 s.closed=true
 for _,l:=range s.listeners{_ = l.Close()}
 s.mu.Unlock()
 s.wg.Wait()
}
func (s *Server) Addrs()[]string{
 s.mu.Lock();defer s.mu.Unlock()
 result:=make([]string,0,len(s.listeners))
 for _,l:=range s.listeners{result=append(result,l.Addr().String())}
 return result
}
func (s *Server) handleSOCKS(c net.Conn){
 header:=make([]byte,2)
 if _,e:=io.ReadFull(c,header);e!=nil||header[0]!=5{return}
 methods:=make([]byte,int(header[1]))
 if _,e:=io.ReadFull(c,methods);e!=nil{return}
 auth:=false
 for _,m:=range methods{if m==2{auth=true}}
 if !auth{c.Write([]byte{5,255});return}
 c.Write([]byte{5,2})
 h:=make([]byte,2)
 if _,e:=io.ReadFull(c,h);e!=nil||h[0]!=1{return}
 user:=make([]byte,int(h[1]));if _,e:=io.ReadFull(c,user);e!=nil{return}
 if _,e:=io.ReadFull(c,h[:1]);e!=nil{return}
 pass:=make([]byte,int(h[0]));if _,e:=io.ReadFull(c,pass);e!=nil{return}
 if string(user)!=s.cfg.Username||string(pass)!=s.cfg.Password{c.Write([]byte{1,1});return}
 c.Write([]byte{1,0})
 req:=make([]byte,4)
 if _,e:=io.ReadFull(c,req);e!=nil||req[0]!=5{return}
 if req[1]!=1{c.Write([]byte{5,7,0,1,0,0,0,0,0,0});return}
 host,e:=readAddr(c,req[3]);if e!=nil{return}
 p:=make([]byte,2);if _,e=io.ReadFull(c,p);e!=nil{return}
 dest:=net.JoinHostPort(host,strconv.Itoa(int(binary.BigEndian.Uint16(p))))
 up,e:=s.dial(context.Background(),dest)
 if e!=nil{c.Write([]byte{5,5,0,1,0,0,0,0,0,0});return}
 defer up.Close()
 c.Write([]byte{5,0,0,1,0,0,0,0,0,0})
 relayBuffered(c,c,up)
}
func readAddr(r io.Reader,atyp byte)(string,error){
 switch atyp{
 case 1:
  b:=make([]byte,4);_,e:=io.ReadFull(r,b);return net.IP(b).String(),e
 case 4:
  b:=make([]byte,16);_,e:=io.ReadFull(r,b);return net.IP(b).String(),e
 case 3:
  n:=make([]byte,1);if _,e:=io.ReadFull(r,n);e!=nil{return "",e}
  b:=make([]byte,int(n[0]));_,e:=io.ReadFull(r,b);return string(b),e
 }
 return "",errors.New("unsupported address type")
}
func (s *Server) handleHTTP(c net.Conn){
 reader:=bufio.NewReader(io.LimitReader(c,65536))
 req,e:=http.ReadRequest(reader);if e!=nil{return}
 defer req.Body.Close()
 if req.Method!="CONNECT"{io.WriteString(c,"HTTP/1.1 405 Method Not Allowed\r\nContent-Length: 0\r\n\r\n");return}
 // CONNECT requires Proxy-Authorization, not origin Authorization.
 h:=req.Header.Get("Proxy-Authorization")
 user,pass,ok:="","",false
 if strings.HasPrefix(h,"Basic "){
  b,e:=base64.StdEncoding.DecodeString(strings.TrimSpace(strings.TrimPrefix(h,"Basic ")))
  if e==nil{user,pass,ok=strings.Cut(string(b),":")}
 }
 if !ok||user!=s.cfg.Username||pass!=s.cfg.Password{
  io.WriteString(c,"HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=Making\r\nContent-Length: 0\r\n\r\n");return
 }
 dest:=req.Host
 if _,_,e=net.SplitHostPort(dest);e!=nil{io.WriteString(c,"HTTP/1.1 400 Bad Request\r\n\r\n");return}
 up,e:=s.dial(context.Background(),dest)
 if e!=nil{io.WriteString(c,"HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n");return}
 defer up.Close()
 io.WriteString(c,"HTTP/1.1 200 Connection Established\r\n\r\n")
 relayBuffered(c,reader,up)
}
func relayBuffered(a net.Conn,from io.Reader,b net.Conn){
 done:=make(chan struct{})
 go func(){io.Copy(b,from);b.Close();close(done)}()
 io.Copy(a,b)
 a.Close();b.Close();<-done
}
func (s *Server) dial(ctx context.Context,dest string)(net.Conn,error){
 u,_:=url.Parse(s.cfg.Upstream)
 dialer:=net.Dialer{Timeout:12*time.Second}
 conn,e:=dialer.DialContext(ctx,"tcp",u.Host);if e!=nil{return nil,e}
 _=conn.SetDeadline(time.Now().Add(12*time.Second))
 if u.Scheme=="socks5"{e=dialSocks(conn,u,dest)}else{e=dialHTTP(conn,u,dest)}
 if e!=nil{conn.Close();return nil,e}
 _=conn.SetDeadline(time.Time{})
 return conn,nil
}
func dialSocks(c net.Conn,u *url.URL,dest string)error{
 methods:=[]byte{5,1,0}
 if u.User!=nil{methods=[]byte{5,1,2}}
 if _,e:=c.Write(methods);e!=nil{return e}
 b:=make([]byte,2);if _,e:=io.ReadFull(c,b);e!=nil{return e}
 if b[0]!=5{return errors.New("upstream SOCKS protocol mismatch")}
 if b[1]==2{
  if u.User==nil{return errors.New("upstream authentication required")}
  user:=u.User.Username();pass,_:=u.User.Password()
  if len(user)>255||len(pass)>255{return errors.New("upstream credentials too long")}
  pkt:=append([]byte{1,byte(len(user))},[]byte(user)...)
  pkt=append(pkt,byte(len(pass)));pkt=append(pkt,[]byte(pass)...)
  if _,e:=c.Write(pkt);e!=nil{return e}
  if _,e:=io.ReadFull(c,b);e!=nil{return e}
  if b[1]!=0{return errors.New("upstream authentication rejected")}
 }else if b[1]!=0{return errors.New("upstream method rejected")}
 host,port,e:=net.SplitHostPort(dest);if e!=nil{return e}
 n,e:=strconv.Atoi(port);if e!=nil||n<1||n>65535{return errors.New("invalid destination port")}
 pkt:=[]byte{5,1,0}
 ip:=net.ParseIP(host)
 if v4:=ip.To4();v4!=nil{pkt=append(pkt,1);pkt=append(pkt,v4...)}else if ip!=nil{pkt=append(pkt,4);pkt=append(pkt,ip.To16()...)}else{
  if len(host)>255{return errors.New("domain too long")}
  pkt=append(pkt,3,byte(len(host)));pkt=append(pkt,[]byte(host)...)
 }
 pkt=binary.BigEndian.AppendUint16(pkt,uint16(n))
 if _,e=c.Write(pkt);e!=nil{return e}
 head:=make([]byte,4);if _,e=io.ReadFull(c,head);e!=nil{return e}
 if head[0]!=5||head[1]!=0{return fmt.Errorf("upstream SOCKS connect rejected (%d)",head[1])}
 if _,e=readAddr(c,head[3]);e!=nil{return e}
 _,e=io.ReadFull(c,b);return e
}
func dialHTTP(c net.Conn,u *url.URL,dest string)error{
 auth:=""
 if u.User!=nil{
  user:=u.User.Username();pass,_:=u.User.Password()
  auth="Proxy-Authorization: Basic "+base64.StdEncoding.EncodeToString([]byte(user+":"+pass))+"\r\n"
 }
 if _,e:=io.WriteString(c,"CONNECT "+dest+" HTTP/1.1\r\nHost: "+dest+"\r\n"+auth+"\r\n");e!=nil{return e}
 reader:=bufio.NewReader(c)
 resp,e:=http.ReadResponse(reader,&http.Request{Method:"CONNECT"});if e!=nil{return e}
 defer resp.Body.Close()
 if resp.StatusCode!=200{return fmt.Errorf("upstream HTTP CONNECT returned %d",resp.StatusCode)}
 if reader.Buffered()>0{return errors.New("upstream sent unexpected bytes after CONNECT response")}
 return nil
}
