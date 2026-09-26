package directory

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/gotify/server/v3/auth/password"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/security"
)

type Database interface {
	GetDirectoryConfig() (*model.DirectoryConfig, error)
	GetUserByName(name string) (*model.User, error)
	CreateUser(user *model.User) error
	UpdateUser(user *model.User) error
}

type Service struct {
	DB       Database
	Strength int
}

type identity struct {
	DN          string
	Username    string
	DisplayName string
	MemberOf    []string
}

func (s *Service) Enabled() bool {
	if s == nil || s.DB == nil { return false }
	config, err := s.DB.GetDirectoryConfig()
	return err == nil && config != nil && config.Enabled
}

func (s *Service) Authenticate(username, plainPassword string) (*model.User, error) {
	if s == nil || s.DB == nil { return nil, errors.New("directory authentication is unavailable") }
	config, err := s.DB.GetDirectoryConfig()
	if err != nil { return nil, err }
	if config == nil || !config.Enabled { return nil, errors.New("directory authentication is disabled") }
	if strings.TrimSpace(username) == "" || plainPassword == "" {
		return nil, errors.New("directory username and password are required")
	}
	identity, err := authenticate(config, username, plainPassword)
	if err != nil { return nil, err }

	user, err := s.DB.GetUserByName(identity.Username)
	if err != nil { return nil, err }
	if user == nil {
		if !config.AutoRegister { return nil, errors.New("directory user is not registered") }
		randomPassword := make([]byte, 32)
		if _, err := rand.Read(randomPassword); err != nil { return nil, err }
		hash, err := password.CreatePassword(fmt.Sprintf("%x", randomPassword), s.Strength)
		if err != nil { return nil, err }
		user = &model.User{
			Name: identity.Username,
			DisplayName: identity.DisplayName,
			Pass: hash,
			Admin: memberOf(identity.MemberOf, config.AdminGroupDN),
			DirectoryManaged: true,
		}
		if err := s.DB.CreateUser(user); err != nil { return nil, err }
		return user, nil
	}

	if !user.DirectoryManaged && !config.LinkByUsername {
		return nil, errors.New("a local account already uses this username and directory linking is disabled")
	}
	user.DirectoryManaged = true
	if strings.TrimSpace(identity.DisplayName) != "" { user.DisplayName = identity.DisplayName }
	if strings.TrimSpace(config.AdminGroupDN) != "" {
		user.Admin = memberOf(identity.MemberOf, config.AdminGroupDN)
	}
	if err := s.DB.UpdateUser(user); err != nil { return nil, err }
	return user, nil
}

func (s *Service) Test(config *model.DirectoryConfig, username, plainPassword string) error {
	if config == nil { return errors.New("directory configuration is required") }
	_, err := authenticate(config, username, plainPassword)
	return err
}

func authenticate(config *model.DirectoryConfig, username, plainPassword string) (*identity, error) {
	conn, err := dial(config)
	if err != nil { return nil, err }
	defer conn.Close()
	client := &ldapClient{conn: conn, reader: bufio.NewReader(conn), nextID: 1}

	bindPassword, err := security.Reveal(config.BindPassword)
	if err != nil { return nil, fmt.Errorf("directory bind password could not be decrypted: %w", err) }
	if strings.TrimSpace(config.BindDN) != "" {
		if err := client.bind(config.BindDN, bindPassword); err != nil {
			return nil, fmt.Errorf("directory service bind failed: %w", err)
		}
	}

	attribute := strings.TrimSpace(config.UserAttribute)
	if attribute == "" { attribute = "uid" }
	displayAttribute := strings.TrimSpace(config.DisplayNameAttribute)
	if displayAttribute == "" { displayAttribute = "displayName" }
	entry, err := client.searchUser(config.UserBaseDN, attribute, username, []string{attribute, displayAttribute, "memberOf"})
	if err != nil { return nil, err }
	if entry == nil { return nil, errors.New("directory user was not found") }

	userConn, err := dial(config)
	if err != nil { return nil, err }
	defer userConn.Close()
	userClient := &ldapClient{conn:userConn, reader:bufio.NewReader(userConn), nextID:1}
	if err := userClient.bind(entry.DN, plainPassword); err != nil {
		return nil, errors.New("directory credentials are invalid")
	}

	resolvedUsername := first(entry.Attributes[attribute])
	if resolvedUsername == "" { resolvedUsername = username }
	return &identity{
		DN: entry.DN,
		Username: resolvedUsername,
		DisplayName: first(entry.Attributes[displayAttribute]),
		MemberOf: entry.Attributes["memberOf"],
	}, nil
}

func dial(config *model.DirectoryConfig) (net.Conn, error) {
	parsed, err := url.Parse(strings.TrimSpace(config.URL))
	if err != nil { return nil, err }
	if parsed.Scheme != "ldap" && parsed.Scheme != "ldaps" {
		return nil, errors.New("directory URL must use ldap:// or ldaps://")
	}
	host := parsed.Host
	if !strings.Contains(host, ":") {
		if parsed.Scheme == "ldaps" { host += ":636" } else { host += ":389" }
	}
	dialer := &net.Dialer{Timeout: 10*time.Second}
	tlsConfig, err := directoryTLSConfig(parsed.Hostname(), config.CACertificatePEM)
	if err != nil { return nil, err }
	if parsed.Scheme == "ldaps" {
		return tls.DialWithDialer(dialer, "tcp", host, tlsConfig)
	}
	conn, err := dialer.Dial("tcp", host)
	if err != nil { return nil, err }
	if !config.StartTLS { return conn, nil }
	client := &ldapClient{conn:conn,reader:bufio.NewReader(conn),nextID:1}
	if err := client.startTLS(); err != nil { conn.Close(); return nil, err }
	tlsConn := tls.Client(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil { conn.Close(); return nil, err }
	return tlsConn, nil
}

func directoryTLSConfig(serverName, caPEM string) (*tls.Config, error) {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil { pool = x509.NewCertPool() }
	if strings.TrimSpace(caPEM) != "" && !pool.AppendCertsFromPEM([]byte(caPEM)) {
		return nil, errors.New("directory CA certificate could not be parsed")
	}
	return &tls.Config{MinVersion:tls.VersionTLS12,ServerName:serverName,RootCAs:pool}, nil
}

func memberOf(groups []string, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" { return false }
	for _, group := range groups {
		if strings.EqualFold(strings.TrimSpace(group), expected) { return true }
	}
	return false
}

func first(values []string) string {
	if len(values)==0 { return "" }
	return values[0]
}

type searchEntry struct {
	DN string
	Attributes map[string][]string
}

type ldapClient struct {
	conn net.Conn
	reader *bufio.Reader
	nextID int
}

func (c *ldapClient) startTLS() error {
	id:=c.nextMessageID()
	request := ldapMessage(id, packet(0x77, packet(0x80, []byte("1.3.6.1.4.1.1466.20037"))))
	if _,err:=c.conn.Write(request); err!=nil{return err}
	_,op,err:=c.readMessage()
	if err!=nil{return err}
	if op.tag!=0x78{return errors.New("directory did not return a StartTLS response")}
	code,_,err:=parseLDAPResult(op.content)
	if err!=nil{return err}
	if code!=0{return fmt.Errorf("directory StartTLS failed with result code %d",code)}
	return nil
}

func (c *ldapClient) bind(dn, secret string) error {
	id:=c.nextMessageID()
	content:=append(integerTLV(3), octetTLV(dn)...)
	content=append(content, packet(0x80,[]byte(secret))...)
	if _,err:=c.conn.Write(ldapMessage(id,packet(0x60,content)));err!=nil{return err}
	_,op,err:=c.readMessage()
	if err!=nil{return err}
	if op.tag!=0x61{return errors.New("directory returned an unexpected bind response")}
	code,diagnostic,err:=parseLDAPResult(op.content)
	if err!=nil{return err}
	if code!=0 {
		if diagnostic!="" { return fmt.Errorf("LDAP result %d: %s",code,diagnostic) }
		return fmt.Errorf("LDAP result %d",code)
	}
	return nil
}

func (c *ldapClient) searchUser(baseDN, attribute, username string, attributes []string) (*searchEntry,error) {
	if strings.TrimSpace(baseDN)=="" {return nil,errors.New("directory user base DN is required")}
	id:=c.nextMessageID()
	content:=append(octetTLV(baseDN), enumeratedTLV(2)...)
	content=append(content,enumeratedTLV(0)...)
	content=append(content,integerTLV(2)...)
	content=append(content,integerTLV(10)...)
	content=append(content,booleanTLV(false)...)
	filterContent:=append(octetTLV(attribute),octetTLV(username)...)
	content=append(content,packet(0xa3,filterContent)...)
	var attrs []byte
	for _,attribute:=range attributes{attrs=append(attrs,octetTLV(attribute)...)}
	content=append(content,packet(0x30,attrs)...)
	if _,err:=c.conn.Write(ldapMessage(id,packet(0x63,content)));err!=nil{return nil,err}

	var found *searchEntry
	for {
		responseID,op,err:=c.readMessage()
		if err!=nil{return nil,err}
		if responseID!=id{continue}
		switch op.tag{
		case 0x64:
			entry,err:=parseSearchEntry(op.content)
			if err!=nil{return nil,err}
			if found!=nil{return nil,errors.New("directory search returned more than one matching user")}
			found=entry
		case 0x65:
			code,diagnostic,err:=parseLDAPResult(op.content)
			if err!=nil{return nil,err}
			if code!=0 {
				if diagnostic!=""{return nil,fmt.Errorf("directory search failed with result %d: %s",code,diagnostic)}
				return nil,fmt.Errorf("directory search failed with result %d",code)
			}
			return found,nil
		}
	}
}

func (c *ldapClient) nextMessageID() int { id:=c.nextID;c.nextID++;return id }

type tlv struct{tag byte;content []byte}

func (c *ldapClient) readMessage()(int,tlv,error){
	tag,content,err:=readNetworkTLV(c.reader)
	if err!=nil{return 0,tlv{},err}
	if tag!=0x30{return 0,tlv{},errors.New("directory returned an invalid LDAP message")}
	offset:=0
	idTLV,err:=readTLV(content,&offset)
	if err!=nil||idTLV.tag!=0x02{return 0,tlv{},errors.New("directory message id is invalid")}
	id:=decodeInteger(idTLV.content)
	op,err:=readTLV(content,&offset)
	return id,op,err
}

func parseLDAPResult(content []byte)(int,string,error){
	offset:=0
	codeTLV,err:=readTLV(content,&offset)
	if err!=nil||codeTLV.tag!=0x0a{return 0,"",errors.New("directory result code is invalid")}
	code:=decodeInteger(codeTLV.content)
	if _,err=readTLV(content,&offset);err!=nil{return 0,"",err}
	message,err:=readTLV(content,&offset)
	if err!=nil{return 0,"",err}
	return code,string(message.content),nil
}

func parseSearchEntry(content []byte)(*searchEntry,error){
	offset:=0
	dn,err:=readTLV(content,&offset)
	if err!=nil||dn.tag!=0x04{return nil,errors.New("directory search DN is invalid")}
	attrSeq,err:=readTLV(content,&offset)
	if err!=nil||attrSeq.tag!=0x30{return nil,errors.New("directory search attributes are invalid")}
	entry:=&searchEntry{DN:string(dn.content),Attributes:make(map[string][]string)}
	attrOffset:=0
	for attrOffset<len(attrSeq.content){
		attr,err:=readTLV(attrSeq.content,&attrOffset)
		if err!=nil||attr.tag!=0x30{return nil,errors.New("directory attribute is invalid")}
		partOffset:=0
		nameTLV,err:=readTLV(attr.content,&partOffset)
		if err!=nil{return nil,err}
		valuesTLV,err:=readTLV(attr.content,&partOffset)
		if err!=nil||valuesTLV.tag!=0x31{return nil,errors.New("directory attribute values are invalid")}
		valueOffset:=0
		for valueOffset<len(valuesTLV.content){
			value,err:=readTLV(valuesTLV.content,&valueOffset)
			if err!=nil{return nil,err}
			entry.Attributes[string(nameTLV.content)]=append(entry.Attributes[string(nameTLV.content)],string(value.content))
		}
	}
	return entry,nil
}

func ldapMessage(id int,op []byte)[]byte{
	content:=append(integerTLV(id),op...)
	return packet(0x30,content)
}

func packet(tag byte,content []byte)[]byte{
	result:=[]byte{tag}
	result=append(result,encodeLength(len(content))...)
	result=append(result,content...)
	return result
}
func integerTLV(value int)[]byte{
	if value==0{return packet(0x02,[]byte{0})}
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:],uint64(value))
	start:=0
	for start<len(buf)-1&&buf[start]==0{start++}
	data:=buf[start:]
	if data[0]&0x80!=0{data=append([]byte{0},data...)}
	return packet(0x02,data)
}
func enumeratedTLV(value int)[]byte{p:=integerTLV(value);p[0]=0x0a;return p}
func booleanTLV(value bool)[]byte{if value{return packet(0x01,[]byte{0xff})};return packet(0x01,[]byte{0})}
func octetTLV(value string)[]byte{return packet(0x04,[]byte(value))}
func encodeLength(length int)[]byte{
	if length<128{return []byte{byte(length)}}
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:],uint64(length))
	start:=0
	for start<len(raw)&&raw[start]==0{start++}
	count:=len(raw)-start
	return append([]byte{0x80|byte(count)},raw[start:]...)
}

func readNetworkTLV(reader *bufio.Reader)(byte,[]byte,error){
	tag,err:=reader.ReadByte()
	if err!=nil{return 0,nil,err}
	length,err:=readNetworkLength(reader)
	if err!=nil{return 0,nil,err}
	if length<0||length>16<<20{return 0,nil,errors.New("directory response exceeds 16 MiB")}
	content:=make([]byte,length)
	_,err=io.ReadFull(reader,content)
	return tag,content,err
}

func readNetworkLength(reader *bufio.Reader)(int,error){
	first,err:=reader.ReadByte()
	if err!=nil{return 0,err}
	if first&0x80==0{return int(first),nil}
	count:=int(first&0x7f)
	if count==0||count>4{return 0,errors.New("invalid BER length")}
	raw:=make([]byte,count)
	if _,err:=io.ReadFull(reader,raw);err!=nil{return 0,err}
	length:=0
	for _,value:=range raw{length=length<<8|int(value)}
	return length,nil
}

func readTLV(data []byte,offset *int)(tlv,error){
	if *offset>=len(data){return tlv{},io.ErrUnexpectedEOF}
	tag:=data[*offset];*offset++
	length,err:=readLength(data,offset)
	if err!=nil{return tlv{},err}
	if length<0||*offset+length>len(data){return tlv{},io.ErrUnexpectedEOF}
	content:=data[*offset:*offset+length]
	*offset+=length
	return tlv{tag:tag,content:content},nil
}
func readLength(data []byte,offset *int)(int,error){
	if *offset>=len(data){return 0,io.ErrUnexpectedEOF}
	first:=data[*offset];*offset++
	if first&0x80==0{return int(first),nil}
	count:=int(first&0x7f)
	if count==0||count>4||*offset+count>len(data){return 0,errors.New("invalid BER length")}
	length:=0
	for i:=0;i<count;i++{length=length<<8|int(data[*offset]);*offset++}
	return length,nil
}
func decodeInteger(data []byte)int{value:=0;for _,b:=range data{value=value<<8|int(b)};return value}

