// +build functest

package functest

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/testutils/launchnet"
)

func customSend(data string) (map[string]interface{}, error) {
	req, err := http.NewRequest("POST", launchnet.TestRPCUrl, strings.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c := http.Client{}
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	err = json.Unmarshal(body, &out)
	return out, err
}

func TestEmptyBody(t *testing.T) {
	res, err := customSend("")
	require.NoError(t, err)
	require.Equal(t, "The JSON received is not a valid request payload.", res["error"].(map[string]interface{})["message"].(string))
}

func TestCrazyJSON(t *testing.T) {
	res, err := customSend("[dh")
	require.NoError(t, err)
	require.Equal(t, res["error"].(map[string]interface{})["message"].(string), "The JSON received is not a valid request payload.")
}
