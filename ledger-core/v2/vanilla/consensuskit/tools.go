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

package consensuskit

// BftMajority function guarantees that (float(bftMajorityCount)/nodeCount > 2.0/3.0)
//  AND	(float(bftMajorityCount - 1)/nodeCount <= 2.0/3.0)
func BftMajority(nodeCount int) int {
	return nodeCount - BftMinority(nodeCount)
}

func BftMinority(nodeCount int) int {
	return (nodeCount - 1) / 3
}
