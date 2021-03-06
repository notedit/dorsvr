package auth

import (
	"crypto/md5"
	"fmt"
	"io"
	"strings"
	sys "syscall"
)

var counter = 0

type Digest struct {
	Realm    string
	Nonce    string
	Username string
	Password string
}

func NewDigest() *Digest {
	return &Digest{}
}

func (d *Digest) RandomNonce() {
	var timeNow sys.Timeval
	sys.Gettimeofday(&timeNow)

	counter++
	seedData := fmt.Sprintf("%d.%06d%d", timeNow.Sec, timeNow.Usec, counter)

	// Use MD5 to compute a 'random' nonce from this seed data:
	h := md5.New()
	io.WriteString(h, seedData)
	d.Nonce = string(h.Sum(nil))
}

func (d *Digest) ComputeResponse(cmd, url string) string {
	ha1Data := fmt.Sprintf("%s:%s:%s", d.Username, d.Realm, d.Password)
	ha2Data := fmt.Sprintf("%s:%s", cmd, url)

	h1 := md5.New()
	h2 := md5.New()
	io.WriteString(h1, ha1Data)
	io.WriteString(h2, ha2Data)

	digestData := fmt.Sprintf("%s:%s:%s", h1.Sum(nil), d.Nonce, h2.Sum(nil))

	h3 := md5.New()
	io.WriteString(h3, digestData)

	return string(h3.Sum(nil))
}

type AuthorizationHeader struct {
	Uri      string
	Realm    string
	Nonce    string
	Username string
	Response string
}

func ParseAuthorizationHeader(buf string) *AuthorizationHeader {
	// First, find "Authorization:"
	for {
		if buf == "" {
			return nil
		}

		if strings.EqualFold(buf[:22], "Authorization: Digest ") {
			break
		}
		buf = buf[1:]
	}

	// Then, run through each of the fields, looking for ones we handle:
	var n1, n2 int
	var parameter, value, username, realm, nonce, uri, response string
	fields := buf[22:]
	for {
		n1, _ = fmt.Sscanf(fields, "%[^=]=\"%[^\"]\"", &parameter, &value)
		n2, _ = fmt.Sscanf(fields, "%[^=]=\"\"", &parameter)
		if n1 != 2 && n2 != 1 {
			break
		}
		if strings.EqualFold(parameter, "username") {
			username = value
		} else if strings.EqualFold(parameter, "realm") {
			realm = value
		} else if strings.EqualFold(parameter, "nonce") {
			nonce = value
		} else if strings.EqualFold(parameter, "uri") {
			uri = value
		} else if strings.EqualFold(parameter, "response") {
			response = value
		}
		fields = fields[len(parameter)+2+len(value)+1:]
		for fields[0] == ' ' || fields[0] == ',' {
			fields = fields[1:]
		}
		if fields == "" || fields[0] == '\r' || fields[0] == '\n' {
			break
		}
	}

	return &AuthorizationHeader{
		Uri:      uri,
		Realm:    realm,
		Nonce:    nonce,
		Username: username,
		Response: response,
	}
}
