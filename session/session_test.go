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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/candango/httpok/security"
	"github.com/stretchr/testify/assert"
)

func chFileTime(name string, mTime time.Time) error {
	err := os.Chtimes(name, mTime, mTime)
	if err != nil {
		return err
	}
	return nil
}

// TODO: We need to keep building the session engine tests
func TestSessionServer(t *testing.T) {
	t.Run("Session store/read with json encoder", func(t *testing.T) {
		engine := NewFileEngine()
		err := engine.Start(context.Background())
		if err != nil {
			t.Error(err)
		}

		sess := security.RandomString(64)

		err = engine.Store(sess, map[string]any{
			"key": "value",
		})
		if err != nil {
			t.Error(err)
		}
		var sessData map[string]any

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
	})

	t.Run("Session purge", func(t *testing.T) {
		engine := NewFileEngine()
		err := engine.Start(context.Background())
		if err != nil {
			t.Error(err)
		}

		sess1 := security.RandomString(64)
		file1 := fmt.Sprintf("%s.sess", filepath.Join(engine.Dir, sess1))
		sess2 := security.RandomString(64)
		file2 := fmt.Sprintf("%s.sess", filepath.Join(engine.Dir, sess2))
		sess3 := security.RandomString(64)
		file3 := fmt.Sprintf("%s.sess", filepath.Join(engine.Dir, sess3))

		err = engine.Store(sess1, map[string]any{
			"key": "value",
		})
		assert.True(t, fileExists(file1))
		err = engine.Store(sess2, map[string]any{
			"key": "value",
		})
		assert.True(t, fileExists(file2))
		err = engine.Store(sess3, map[string]any{
			"key": "value",
		})
		assert.True(t, fileExists(file3))
		if err != nil {
			t.Error(err)
		}

		oldTime := time.Date(2023, time.October, 1, 12, 0, 0, 0, time.UTC)
		err = chFileTime(file1, oldTime)
		if err != nil {
			t.Error(err)
		}
		err = chFileTime(file2, oldTime)
		if err != nil {
			t.Error(err)
		}

		engine.Purge()

		assert.False(t, fileExists(file1))
		assert.False(t, fileExists(file2))
		assert.True(t, fileExists(file3))

		err = chFileTime(file3, oldTime)
		if err != nil {
			t.Error(err)
		}
		engine.Purge()
		assert.False(t, fileExists(file3))
	})
}
