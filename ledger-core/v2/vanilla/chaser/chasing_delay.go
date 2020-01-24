//
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

package chaser

import "time"

func NewChasingTimer(chasingDelay time.Duration) ChasingTimer {
	return ChasingTimer{chasingDelay: chasingDelay}
}

type ChasingTimer struct {
	chasingDelay time.Duration
	timer        *time.Timer
	wasCleared   bool
}

func (c *ChasingTimer) IsEnabled() bool {
	return c.chasingDelay > 0
}

func (c *ChasingTimer) WasStarted() bool {
	return c.timer != nil
}

func (c *ChasingTimer) RestartChase() {

	if c.chasingDelay <= 0 {
		return
	}

	if c.timer == nil {
		c.timer = time.NewTimer(c.chasingDelay)
		return
	}

	// Restart chasing timer from this moment
	if !c.wasCleared && !c.timer.Stop() {
		<-c.timer.C
	}
	c.wasCleared = false
	c.timer.Reset(c.chasingDelay)
}

func (c *ChasingTimer) Channel() <-chan time.Time {
	if c.timer == nil {
		return nil // receiver will wait indefinitely
	}
	return c.timer.C
}

func (c *ChasingTimer) ClearExpired() {
	c.wasCleared = true
}
