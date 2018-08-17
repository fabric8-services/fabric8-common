package jwk_test

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/fabric8-services/fabric8-common/test"
	testsuite "github.com/fabric8-services/fabric8-common/test/suite"
	"github.com/fabric8-services/fabric8-common/token/jwk"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestFetchKeysSuite struct {
	testsuite.UnitTestSuite
}

func TestRunFetchKeysSuite(t *testing.T) {
	suite.Run(t, &TestFetchKeysSuite{UnitTestSuite: testsuite.NewUnitTestSuite()})
}

func (s *TestFetchKeysSuite) TestDefaultFetcher() {
	require.NotPanics(s.T(), func() { jwk.FetchKeys("") })
}

func (s *TestFetchKeysSuite) TestFetchKeys() {
	client := &test.DummyHttpClient{AssertRequest: func(req *http.Request) {
		assert.Equal(s.T(), "GET", req.Method)
		assert.Equal(s.T(), "https://openshift.io/keys", req.URL.String())
	}}
	keyLoader := jwk.KeyLoader{HttpClient: client}
	client.Response = responseOK()

	// All three keys are loaded
	loadedKeys, err := keyLoader.FetchKeys("https://openshift.io/keys")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), loadedKeys)
	require.Len(s.T(), loadedKeys, 3)
	expectedKeys := map[string]string{
		"aUGv8mQA85jg4V1DU8Uk1W0uKsxn187KQONAGl6AMtc": "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA40yB6SNoU4SpWxTfG5ilu+BlLYikRyyEcJIGg//w/GyqtjvT/CVo92DRTh/DlrgwjSitmZrhauBnrCOoUBMin0/TXeSo3w2M5tEiiIFPbTDRf2jMfbSGEOke9O0USCCR+bM2TncrgZR74qlSwq38VCND4zHc89rAzqJ2LVM2aXkuBbO7TcgLNyooBrpOK9khVHAD64cyODAdJY4esUjcLdlcB7TMDGOgxGGn2RARU7+TUf32gZZbTMikbuPM5gXuzGlo/22ECbQSKuZpbGwgPIAZ5NN9QA4D1NRz9+KDoiXZ6deZTTVCrZykJJ6RyLNfRh+XS+6G5nvcqAmfBpyOWwIDAQAB",
		"9MLnViaRkhVj1GT9kpWUkwHIwUD-wZfUxR-3CpkE-Xs": "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAnwrjH5iTSErw9xUptp6QSFoUfpHUXZ+PaslYSUrpLjw1q27ODSFwmhV4+dAaTMO5chFv/kM36H3ZOyA146nwxBobS723okFaIkshRrf6qgtD6coTHlVUSBTAcwKEjNn4C9jtEpyOl+eSgxhMzRH3bwTIFlLlVMiZf7XVE7P3yuOCpqkk2rdYVSpQWQWKU+ZRywJkYcLwjEYjc70AoNpjO5QnY+Exx98E30iEdPHZpsfNhsjh9Z7IX5TrMYgz7zBTw8+niO/uq3RBaHyIhDbvenbR9Q59d88lbnEeHKgSMe2RQpFR3rxFRkc/64Rn/bMuL/ptNowPqh1P+9GjYzWmPwIDAQAB",
		"bNq-BCOR3ev-E6buGSaPrU-0SXX8whhDlmZ6geenkTE": "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvQ8p+HsTMrgcsuIMoOR1LXRhynL9YAU0qoDON6PLKCpdBv0Xy/jnsPjo5DrtUOijuJcID8CR7E0hYpY9MgK5H5pDFwC4lbUVENquHEVS/E0pQSKCIzSmORcIhjYW2+wKfDOVjeudZwdFBIxJ6KpIty/aF78hlUJZuvghFVqoHQYTq/DZOmKjS+PAVLw8FKE3wa/3WU0EkpP+iovRMCkllzxqrcLPIvx+T2gkwe0bn0kTvdMOhTLTN2tuvKrFpVUxVi8RM/V8PtgdKroxnES7SyUqK8rLO830jKJzAYrByQL+sdGuSqInIY/geahQHEGTwMI0CLj6zfhpjSgCflstvwIDAQAB",
	}
	for _, key := range loadedKeys {
		pk := expectedKeys[key.KeyID]
		require.NotNil(s.T(), pk)
		require.Equal(s.T(), pk, toPem(s.T(), key.Key))
	}

	// Fail if the client returned an error
	client.Response = responseOK()
	client.Error = errors.New("something went wrong")
	_, err = keyLoader.FetchKeys("https://openshift.io/keys")
	require.Error(s.T(), err)
	assert.Equal(s.T(), err, client.Error)

	// Fail if the client returned an error
	client.Response = responseError()
	client.Error = nil
	_, err = keyLoader.FetchKeys("https://openshift.io/keys")
	require.Error(s.T(), err)
	assert.Equal(s.T(), err.Error(), "unable to obtain public keys from remote service")

	// Fail if the client returned incorrect JSON
	client.Response = responseIncorrectJSON()
	_, err = keyLoader.FetchKeys("https://openshift.io/keys")
	require.Error(s.T(), err)
	assert.Equal(s.T(), err.Error(), "unexpected end of JSON input")
}

func toPem(t *testing.T, key *rsa.PublicKey) string {
	pubASN1, err := x509.MarshalPKIXPublicKey(key)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(pubASN1)
}

func responseOK() *http.Response {
	body := ioutil.NopCloser(bytes.NewReader([]byte(keysJSON)))
	return &http.Response{Body: body, StatusCode: http.StatusOK}
}

func responseError() *http.Response {
	body := ioutil.NopCloser(bytes.NewReader([]byte(keysJSON)))
	return &http.Response{Body: body, StatusCode: http.StatusInternalServerError}
}

func responseIncorrectJSON() *http.Response {
	body := ioutil.NopCloser(bytes.NewReader([]byte("")))
	return &http.Response{Body: body, StatusCode: http.StatusOK}
}

const (
	keysJSON = `{
		        "keys": [
		          {
        		    "alg": "RS256",
		            "e": "AQAB",
        		    "kid": "aUGv8mQA85jg4V1DU8Uk1W0uKsxn187KQONAGl6AMtc",
		            "kty": "RSA",
        		    "n": "40yB6SNoU4SpWxTfG5ilu-BlLYikRyyEcJIGg__w_GyqtjvT_CVo92DRTh_DlrgwjSitmZrhauBnrCOoUBMin0_TXeSo3w2M5tEiiIFPbTDRf2jMfbSGEOke9O0USCCR-bM2TncrgZR74qlSwq38VCND4zHc89rAzqJ2LVM2aXkuBbO7TcgLNyooBrpOK9khVHAD64cyODAdJY4esUjcLdlcB7TMDGOgxGGn2RARU7-TUf32gZZbTMikbuPM5gXuzGlo_22ECbQSKuZpbGwgPIAZ5NN9QA4D1NRz9-KDoiXZ6deZTTVCrZykJJ6RyLNfRh-XS-6G5nvcqAmfBpyOWw",
		            "use": "sig"
        		  },
		          {
        		    "alg": "RS256",
		            "e": "AQAB",
        		    "kid": "9MLnViaRkhVj1GT9kpWUkwHIwUD-wZfUxR-3CpkE-Xs",
		            "kty": "RSA",
        		    "n": "nwrjH5iTSErw9xUptp6QSFoUfpHUXZ-PaslYSUrpLjw1q27ODSFwmhV4-dAaTMO5chFv_kM36H3ZOyA146nwxBobS723okFaIkshRrf6qgtD6coTHlVUSBTAcwKEjNn4C9jtEpyOl-eSgxhMzRH3bwTIFlLlVMiZf7XVE7P3yuOCpqkk2rdYVSpQWQWKU-ZRywJkYcLwjEYjc70AoNpjO5QnY-Exx98E30iEdPHZpsfNhsjh9Z7IX5TrMYgz7zBTw8-niO_uq3RBaHyIhDbvenbR9Q59d88lbnEeHKgSMe2RQpFR3rxFRkc_64Rn_bMuL_ptNowPqh1P-9GjYzWmPw",
		            "use": "sig"
		          },
        		  {
		            "alg": "RS256",
        		    "e": "AQAB",
		            "kid": "bNq-BCOR3ev-E6buGSaPrU-0SXX8whhDlmZ6geenkTE",
        		    "kty": "RSA",
		            "n": "vQ8p-HsTMrgcsuIMoOR1LXRhynL9YAU0qoDON6PLKCpdBv0Xy_jnsPjo5DrtUOijuJcID8CR7E0hYpY9MgK5H5pDFwC4lbUVENquHEVS_E0pQSKCIzSmORcIhjYW2-wKfDOVjeudZwdFBIxJ6KpIty_aF78hlUJZuvghFVqoHQYTq_DZOmKjS-PAVLw8FKE3wa_3WU0EkpP-iovRMCkllzxqrcLPIvx-T2gkwe0bn0kTvdMOhTLTN2tuvKrFpVUxVi8RM_V8PtgdKroxnES7SyUqK8rLO830jKJzAYrByQL-sdGuSqInIY_geahQHEGTwMI0CLj6zfhpjSgCflstvw",
        		    "use": "sig"
		          }
		        ]
      		}`
)
