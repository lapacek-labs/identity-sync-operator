// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package result

import (
	"time"

	controllerruntime "sigs.k8s.io/controller-runtime"
)

type Decision struct {
	RequeueAfter time.Duration
	Outcome      Outcome
	Reason       Reason
	Msg          string
	Err          error
}

func (d Decision) Result() (controllerruntime.Result, error) {
	if d.Err != nil {
		return controllerruntime.Result{}, d.Err
	}
	if d.RequeueAfter > 0 {
		return controllerruntime.Result{RequeueAfter: d.RequeueAfter}, nil
	}
	return controllerruntime.Result{}, nil
}
