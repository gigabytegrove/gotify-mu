package ldapauth

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	URL                   string
	BindDN                string
	BindPassword          string
	BaseDN                string
	UserFilter            string
	DisplayNameAttribute  string
	GroupAttribute        string
	AdminGroupDN          string
	UserGroupDN           string
	CAFile                string
	InsecureSkipVerify    bool
	Timeout               time.Duration
}

type User struct {
	DN          string
	Username    string
	DisplayName string
	Groups      []string
	Admin       bool
}

type tlv struct {
	tag byte
	value []byte
}

func encLen(n int) []byte {
	if n < 128 { return []byte{byte(n)} }
	var raw [8]byte
	i:=len(raw)
	for n>0 { i--;raw[i]=byte(n);n>>=8 }
	out:=[]byte{0x80|byte(len(raw)-i)}
	return append(out,raw[i:]...)
}
func enc(tag byte, content ...[]byte) []byte {
	body:=bytes.Join(content,nil)
	out:=[]byte{tag};out=append(out,encLen(len(body))...);out=append(out,body...);return out
}
func encInt(tag byte, value int) []byte {
	if value==0{return enc(tag,[]byte{0})}
	var raw [8]byte;i:=8;v:=value
	for v>0{i--;raw[i]=byte(v);v>>=8}
	if raw[i]&0x80!=0{i--;raw[i]=0}
	return enc(tag,raw[i:])
}
func encString(value string) []byte { return enc(0x04,[]byte(value)) }
func encBool(value bool) []byte { if value{return enc(0x01,[]byte{0xff})};return enc(0x01,[]byte{0}) }

func readLength(reader *bufio.Reader)(int,error){
	first,err:=reader.ReadByte();if err!=nil{return 0,err}
	if first&0x80==0{return int(first),nil}
	count:=int(first&0x7f);if count==0||count>4{return 0,errors.New("unsupported LDAP length")}
	n:=0
	for i:=0;i<count;i++{b,err:=reader.ReadByte();if err!=nil{return 0,err};n=(n<<8)|int(b)}
	return n,nil
}
func readTLV(reader *bufio.Reader)(tlv,error){
	tag,err:=reader.ReadByte();if err!=nil{return tlv{},err}
	length,err:=readLength(reader);if err!=nil{return tlv{},err}
	if length<0||length>16<<20{return tlv{},errors.New("LDAP response too large")}
	value:=make([]byte,length);_,err=io.ReadFull(reader,value)
	return tlv{tag:tag,value:value},err
}
func children(raw []byte)([]tlv,error){
	reader:=bufio.NewReader(bytes.NewReader(raw));var result []tlv
	for reader.Buffered()>0 || reader.Size()>0 {
		if _,err:=reader.Peek(1);err!=nil {
			if errors.Is(err,io.EOF){break};return nil,err
		}
		item,err:=readTLV(reader);if err!=nil{return nil,err};result=append(result,item)
	}
	return result,nil
}
func intValue(raw []byte) int { n:=0;for _,b:=range raw{n=(n<<8)|int(b)};return n }

type Client struct {
	conn net.Conn
	reader *bufio.Reader
	nextID int
}

func Dial(cfg Config)(*Client,error){
	parsed,err:=url.Parse(strings.TrimSpace(cfg.URL));if err!=nil{return nil,err}
	if parsed.Scheme!="ldap"&&parsed.Scheme!="ldaps"{return nil,errors.New("LDAP URL must use ldap or ldaps")}
	host:=parsed.Hostname();if host==""{return nil,errors.New("LDAP URL must include a host")}
	port:=parsed.Port();if port==""{if parsed.Scheme=="ldaps"{port="636"}else{port="389"}}
	address:=net.JoinHostPort(host,port)
	timeout:=cfg.Timeout;if timeout<=0{timeout=10*time.Second}
	dialer:=&net.Dialer{Timeout:timeout}
	var conn net.Conn
	if parsed.Scheme=="ldaps"{
		tlsCfg:=&tls.Config{MinVersion:tls.VersionTLS12,ServerName:host,InsecureSkipVerify:cfg.InsecureSkipVerify}
		if cfg.CAFile!=""{
			pem,readErr:=os.ReadFile(cfg.CAFile);if readErr!=nil{return nil,readErr}
			pool,certErr:=x509.SystemCertPool();if certErr!=nil||pool==nil{pool=x509.NewCertPool()}
			if !pool.AppendCertsFromPEM(pem){return nil,errors.New("LDAP CA file did not contain a valid certificate")}
			tlsCfg.RootCAs=pool
		}
		conn,err=tls.DialWithDialer(dialer,"tcp",address,tlsCfg)
	}else{conn,err=dialer.Dial("tcp",address)}
	if err!=nil{return nil,err}
	return &Client{conn:conn,reader:bufio.NewReader(conn),nextID:1},nil
}
func (c *Client) Close()error{return c.conn.Close()}
func (c *Client) send(protocol []byte)(int,error){
	id:=c.nextID;c.nextID++
	message:=enc(0x30,encInt(0x02,id),protocol)
	_ = c.conn.SetDeadline(time.Now().Add(15*time.Second))
	_,err:=c.conn.Write(message);return id,err
}
func (c *Client) readMessage()(int,tlv,error){
	outer,err:=readTLV(c.reader);if err!=nil{return 0,tlv{},err}
	if outer.tag!=0x30{return 0,tlv{},errors.New("invalid LDAP message")}
	parts,err:=children(outer.value);if err!=nil||len(parts)<2{return 0,tlv{},errors.New("invalid LDAP message fields")}
	return intValue(parts[0].value),parts[1],nil
}

func (c *Client) Bind(dn,password string)error{
	id,err:=c.send(enc(0x60,encInt(0x02,3),encString(dn),enc(0x80,[]byte(password))))
	if err!=nil{return err}
	responseID,op,err:=c.readMessage();if err!=nil{return err}
	if responseID!=id||op.tag!=0x61{return errors.New("unexpected LDAP bind response")}
	fields,err:=children(op.value);if err!=nil||len(fields)<3{return errors.New("invalid LDAP bind response")}
	if code:=intValue(fields[0].value);code!=0{return fmt.Errorf("LDAP bind failed with result code %d",code)}
	return nil
}

func escapeFilter(value string)string{
	var b strings.Builder
	for _,r:=range []byte(value){
		switch r{case '*','(',')','\\',0:fmt.Fprintf(&b,"\\%02x",r);default:b.WriteByte(r)}
	}
	return b.String()
}

type filterParser struct{ s string; i int }
func (p *filterParser) skip(){for p.i<len(p.s)&&(p.s[p.i]==' '||p.s[p.i]=='\t'||p.s[p.i]=='\n'){p.i++}}
func (p *filterParser) parse()([]byte,error){
	p.skip();if p.i>=len(p.s)||p.s[p.i]!='(' {return nil,errors.New("LDAP filter must begin with (")};p.i++;p.skip()
	if p.i>=len(p.s){return nil,errors.New("invalid LDAP filter")}
	switch p.s[p.i]{
	case '&','|':
		ch:=p.s[p.i];p.i++;var childrenEncoded [][]byte
		for{p.skip();if p.i<len(p.s)&&p.s[p.i]=='(' {child,err:=p.parse();if err!=nil{return nil,err};childrenEncoded=append(childrenEncoded,child);continue};break}
		p.skip();if p.i>=len(p.s)||p.s[p.i]!=')'{return nil,errors.New("unterminated LDAP filter")};p.i++
		if ch=='&'{return enc(0xa0,childrenEncoded...),nil};return enc(0xa1,childrenEncoded...),nil
	case '!':
		p.i++;child,err:=p.parse();if err!=nil{return nil,err};p.skip();if p.i>=len(p.s)||p.s[p.i]!=')'{return nil,errors.New("unterminated LDAP filter")};p.i++;return enc(0xa2,child),nil
	default:
		start:=p.i
		for p.i<len(p.s)&&p.s[p.i]!='='&&p.s[p.i]!=')'{p.i++}
		if p.i>=len(p.s)||p.s[p.i]!='='{return nil,errors.New("only LDAP equality filters are supported")}
		attr:=strings.TrimSpace(p.s[start:p.i]);p.i++;start=p.i
		for p.i<len(p.s)&&p.s[p.i]!=')'{p.i++}
		if p.i>=len(p.s){return nil,errors.New("unterminated LDAP filter")}
		value:=p.s[start:p.i];p.i++
		if attr==""{return nil,errors.New("LDAP filter attribute is empty")}
		return enc(0xa3,encString(attr),encString(value)),nil
	}
}

type Entry struct { DN string; Attributes map[string][]string }

func (c *Client) Search(baseDN,filter string,attributes []string)([]Entry,error){
	parser:=&filterParser{s:filter};encodedFilter,err:=parser.parse();if err!=nil{return nil,err}
	var attrs [][]byte;for _,attr:=range attributes{attrs=append(attrs,encString(attr))}
	request:=enc(0x63,
		encString(baseDN),encInt(0x0a,2),encInt(0x0a,0),encInt(0x02,10),encInt(0x02,10),encBool(false),
		encodedFilter,enc(0x30,attrs...),
	)
	id,err:=c.send(request);if err!=nil{return nil,err}
	var result []Entry
	for{
		responseID,op,err:=c.readMessage();if err!=nil{return nil,err};if responseID!=id{continue}
		switch op.tag{
		case 0x64:
			fields,err:=children(op.value);if err!=nil||len(fields)<2{continue}
			entry:=Entry{DN:string(fields[0].value),Attributes:map[string][]string{}}
			attrList,err:=children(fields[1].value);if err!=nil{continue}
			for _,attrTLV:=range attrList{
				parts,err:=children(attrTLV.value);if err!=nil||len(parts)<2{continue}
				name:=strings.ToLower(string(parts[0].value));vals,err:=children(parts[1].value);if err!=nil{continue}
				for _,v:=range vals{entry.Attributes[name]=append(entry.Attributes[name],string(v.value))}
			}
			result=append(result,entry)
		case 0x65:
			fields,err:=children(op.value);if err!=nil||len(fields)<1{return nil,errors.New("invalid LDAP search result")}
			if code:=intValue(fields[0].value);code!=0{return nil,fmt.Errorf("LDAP search failed with result code %d",code)}
			return result,nil
		}
	}
}

func Authenticate(cfg Config, username, password string)(*User,error){
	if strings.TrimSpace(username)==""||password==""{return nil,errors.New("username and password are required")}
	client,err:=Dial(cfg);if err!=nil{return nil,err};defer client.Close()
	if cfg.BindDN!="" {
		if err:=client.Bind(cfg.BindDN,cfg.BindPassword);err!=nil{return nil,err}
	}
	filter:=cfg.UserFilter;if strings.TrimSpace(filter)==""{filter="(uid={username})"}
	filter=strings.ReplaceAll(filter,"{username}",escapeFilter(username))
	displayAttr:=strings.ToLower(strings.TrimSpace(cfg.DisplayNameAttribute));if displayAttr==""{displayAttr="cn"}
	groupAttr:=strings.ToLower(strings.TrimSpace(cfg.GroupAttribute));if groupAttr==""{groupAttr="memberof"}
	entries,err:=client.Search(cfg.BaseDN,filter,[]string{displayAttr,groupAttr});if err!=nil{return nil,err}
	if len(entries)!=1{return nil,errors.New("directory user was not found or was ambiguous")}
	entry:=entries[0]
	if err:=client.Bind(entry.DN,password);err!=nil{return nil,errors.New("invalid directory credentials")}
	display:=username;if values:=entry.Attributes[displayAttr];len(values)>0&&strings.TrimSpace(values[0])!=""{display=values[0]}
	groups:=entry.Attributes[groupAttr]
	user:=&User{DN:entry.DN,Username:username,DisplayName:display,Groups:groups}
	for _,group:=range groups{
		if cfg.AdminGroupDN!=""&&strings.EqualFold(strings.TrimSpace(group),strings.TrimSpace(cfg.AdminGroupDN)){user.Admin=true}
	}
	if cfg.UserGroupDN!=""&&!user.Admin {
		allowed:=false;for _,group:=range groups{if strings.EqualFold(strings.TrimSpace(group),strings.TrimSpace(cfg.UserGroupDN)){allowed=true;break}}
		if !allowed{return nil,errors.New("directory user is not in an allowed group")}
	}
	return user,nil
}

func ParseBool(value string) bool {
	value=strings.ToLower(strings.TrimSpace(value));result,_:=strconv.ParseBool(value);return result
}
