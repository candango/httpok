// Copyright 2023-2025 Flavio Garcia
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package session

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/candango/httpok/util"
	"github.com/stretchr/testify/assert"
)

// TODO: We need to keep building the session engine tests
func TestSessionServer(t *testing.T) {
	engine := NewFileEngine()
	err := engine.Start()
	if err != nil {
		t.Error(err)
	}

	sess := util.RandomString(16)

	err = engine.Store(sess, map[string]interface{}{
		"key": "value",
	})
	if err != nil {
		t.Error(err)
	}
	var sessData map[string]interface{}

	err = engine.Read(sess, &sessData)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, "value", sessData["key"])
	defer func() {
		sessFile := filepath.Join(engine.Dir, fmt.Sprintf("%s.sess", sess))
		err := os.Remove(sessFile)
		if err != nil {
			t.Error(err)
		}

	}()
}
