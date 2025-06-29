// Copyright 2024 xgfone
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package netx

import (
	"errors"
	"net"
	"testing"
	"time"
)

func TestIPIsOn(t *testing.T) {
	if on, err := IPIsOn(""); err != nil {
		t.Error(err)
	} else if on {
		t.Error("expect empty ip is not on, but got yes")
	}

	if _, err := IPIsOn("abc"); err == nil {
		t.Error("expect an error, but got nil")
	}

	if on, err := IPIsOn("127.0.0.1"); err != nil {
		t.Error(err)
	} else if !on {
		t.Error("expect ip is on, but got not")
	}

	if on, err := IPIsOn("1.2.3.4"); err != nil {
		t.Error(err)
	} else if on {
		t.Error("expect ip is not on, but got yes")
	}
}

func TestIsTimeout(t *testing.T) {
	if IsTimeout(errors.New("false")) {
		t.Error("expect false, but got true")
	}

	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", time.Microsecond)
	if err == nil {
		_ = conn.Close()
		t.Error("expect an error, but got nil")
	} else if !IsTimeout(err) {
		t.Errorf("expect a timeout error, but got '%s'", err.Error())
	}
}

func TestSplitHostPort(t *testing.T) {
	if _, port := SplitHostPort(""); port != "" {
		t.Errorf("expect '%s', but got '%s'", "", port)
	}

	if _, port := SplitHostPort("[abc"); port != "" {
		t.Errorf("expect '%s', but got '%s'", "", port)
	}

	if _, port := SplitHostPort("[abc]"); port != "" {
		t.Errorf("expect '%s', but got '%s'", "", port)
	}

	if _, port := SplitHostPort("[abc]:80"); port != "80" {
		t.Errorf("expect '%s', but got '%s'", "80", port)
	}

	if _, port := SplitHostPort("[ff00::]:80"); port != "80" {
		t.Errorf("expect '%s', but got '%s'", "", port)
	}

	if _, port := SplitHostPort("ff00::"); port != "" {
		t.Errorf("expect '%s', but got '%s'", "", port)
	}

	if _, port := SplitHostPort("1.2.3.4"); port != "" {
		t.Errorf("expect '%s', but got '%s'", "", port)
	}

	if _, port := SplitHostPort("1.2.3.4:80"); port != "80" {
		t.Errorf("expect '%s', but got '%s'", "80", port)
	}

	if _, port := SplitHostPort("localhost"); port != "" {
		t.Errorf("expect '%s', but got '%s'", "", port)
	}

	if _, port := SplitHostPort("localhost:80"); port != "80" {
		t.Errorf("expect '%s', but got '%s'", "80", port)
	}
}
