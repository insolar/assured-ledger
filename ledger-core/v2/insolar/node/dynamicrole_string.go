// Code generated by "stringer -type=DynamicRole"; DO NOT EDIT.

package node

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[DynamicRoleUndefined-0]
	_ = x[DynamicRoleVirtualExecutor-1]
	_ = x[DynamicRoleVirtualValidator-2]
	_ = x[DynamicRoleLightExecutor-3]
	_ = x[DynamicRoleLightValidator-4]
	_ = x[DynamicRoleHeavyExecutor-5]
}

const _DynamicRole_name = "DynamicRoleUndefinedDynamicRoleVirtualExecutorDynamicRoleVirtualValidatorDynamicRoleLightExecutorDynamicRoleLightValidatorDynamicRoleHeavyExecutor"

var _DynamicRole_index = [...]uint8{0, 20, 46, 73, 97, 122, 146}

func (i DynamicRole) String() string {
	if i < 0 || i >= DynamicRole(len(_DynamicRole_index)-1) {
		return "DynamicRole(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _DynamicRole_name[_DynamicRole_index[i]:_DynamicRole_index[i+1]]
}
