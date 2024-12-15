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

package netipx

import (
	"net"
	"testing"
)

type ipstring string

func (s ipstring) Network() string { return "" }
func (s ipstring) String() string  { return string(s) }

func TestFromNetAddr(t *testing.T) {
	expect1 := "192.168.1.1"
	addr1, err := AddrFromNetAddr(&net.TCPAddr{IP: net.IPv4(192, 168, 1, 1).To4()})
	if err != nil {
		t.Error(err)
	} else if s := addr1.String(); s != expect1 {
		t.Errorf("expect ip '%s', but got '%s'", expect1, s)
	}

	expect2 := "192.168.1.2"
	addr2, err := AddrFromNetAddr(&net.UDPAddr{IP: net.IPv4(192, 168, 1, 2).To4()})
	if err != nil {
		t.Error(err)
	} else if s := addr2.String(); s != expect2 {
		t.Errorf("expect ip '%s', but got '%s'", expect2, s)
	}

	expect3 := "192.168.1.3"
	addr3, err := AddrFromNetAddr(ipstring("192.168.1.3:80"))
	if err != nil {
		t.Error(err)
	} else if s := addr3.String(); s != expect3 {
		t.Errorf("expect ip '%s', but got '%s'", expect3, s)
	}

	expect4 := "192.168.1.4"
	addr4, err := AddrFromNetAddr(ipstring("192.168.1.4"))
	if err != nil {
		t.Error(err)
	} else if s := addr4.String(); s != expect4 {
		t.Errorf("expect ip '%s', but got '%s'", expect4, s)
	}

	_, err = AddrFromNetAddr(ipstring("abc"))
	if err == nil {
		t.Errorf("expect a panic, but got nil")
	}
}
