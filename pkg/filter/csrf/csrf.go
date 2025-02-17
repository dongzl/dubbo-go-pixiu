/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package csrf

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	stdHttp "net/http"
)

import (
	"github.com/apache/dubbo-go-pixiu/pkg/common/constant"
	"github.com/apache/dubbo-go-pixiu/pkg/common/extension/filter"
	"github.com/apache/dubbo-go-pixiu/pkg/context/http"
)

const (
	// Kind is the kind of Fallback.
	Kind = constant.HTTPCsrfFilter
)

const (
	csrfSecret = "csrfSecret"
	csrfSalt   = "csrfSalt"
)

func init() {
	filter.RegisterHttpFilter(&Plugin{})
}

type (
	// Plugin is http filter plugin.
	Plugin struct {
	}

	// Filter is http filter instance
	Filter struct {
		cfg *Config
	}

	// Config describe the config of Filter
	Config struct {
		Key           string   `yaml:"key" json:"key" mapstructure:"key"`                                  // get request key
		Secret        string   `yaml:"secret" json:"secret" mapstructure:"secret"`                         // private key
		ErrorMsg      string   `yaml:"error_msg" json:"error_msg" mapstructure:"error_msg"`                // hint error info
		IgnoreMethods []string `yaml:"ignore_methods" json:"ignore_methods" mapstructure:"ignore_methods"` // ignore request method
	}
)

func (p *Plugin) Kind() string {
	return Kind
}

func (p *Plugin) CreateFilter() (filter.HttpFilter, error) {
	return &Filter{cfg: &Config{}}, nil
}

func (f *Filter) PrepareFilterChain(ctx *http.HttpContext) error {
	ctx.AppendFilterFunc(f.Handle)
	return nil
}

func (f *Filter) Handle(ctx *http.HttpContext) {
	f.handleCsrf(ctx)
}

func (f *Filter) handleCsrf(ctx *http.HttpContext) {
	ctx.Request.Header.Set(csrfSecret, f.cfg.Secret)

	if inMethod(f.cfg.IgnoreMethods, ctx.Request.Method) {
		ctx.Next()
		return
	}

	salt := ctx.Request.Header.Get(csrfSalt)

	if salt == "" {
		bt, _ := json.Marshal(http.ErrResponse{Message: f.cfg.ErrorMsg})
		ctx.WriteJSONWithStatus(stdHttp.StatusForbidden, bt)
		ctx.Abort()
		return
	}

	token := tokenize(f.cfg.Secret, salt)

	if token != tokenGetter(ctx, f.cfg.Key) {
		bt, _ := json.Marshal(http.ErrResponse{Message: f.cfg.ErrorMsg})
		ctx.WriteJSONWithStatus(stdHttp.StatusForbidden, bt)
		ctx.Abort()
		return
	}

	ctx.Next()

}

func tokenGetter(ctx *http.HttpContext, key string) string {
	req := ctx.Request
	if t := req.Form.Get(key); t != "" {
		return t
	} else if t := req.URL.Query().Get(key); t != "" {
		return t
	} else if t := req.Header.Get(key); t != "" {
		return t
	}
	return ""
}

func inMethod(methods []string, method string) bool {
	for _, v := range methods {
		if v == method {
			return true
		}
	}
	return false
}

func tokenize(secret, salt string) string {
	return base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s", salt, secret)))
}

func (f *Filter) Apply() error {
	return nil
}

func (f *Filter) Config() interface{} {
	return f.cfg
}
