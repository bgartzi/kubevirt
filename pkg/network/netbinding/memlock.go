/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package netbinding

import (
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
)

func NetBindingHasMemoryLockRequirements(binding *v1.InterfaceBindingPlugin) bool {
	return binding.MemoryLockLimits != nil && (binding.MemoryLockLimits.LockGuestMemory || binding.MemoryLockLimits.Offset != nil && !binding.MemoryLockLimits.Offset.IsZero())
}

func ApplyNetBindingMemlockRequirements(
	guestMemory *resource.Quantity,
	vmi *v1.VirtualMachineInstance,
	registeredPlugins map[string]v1.InterfaceBindingPlugin,
) *resource.Quantity {
	ratio, offset := gatherNetBindingMemLockRequirements(vmi, registeredPlugins)

	if ratio == 0 {
		if offset != nil || offset.IsZero() {
			return guestMemory
		}

		ratio = 1
	}

	res := resource.NewScaledQuantity(guestMemory.ScaledValue(resource.Kilo)*ratio, resource.Kilo)
	res.Add(*offset)

	return res
}

func gatherNetBindingMemLockRequirements(
	vmi *v1.VirtualMachineInstance,
	registeredPlugins map[string]v1.InterfaceBindingPlugin,
) (int64, *resource.Quantity) {
	resOffset := resource.NewScaledQuantity(0, resource.Kilo)
	var resRatio int64 = 0

	for bindingName, count := range netBindingCountMap(vmi, registeredPlugins) {
		binding := registeredPlugins[bindingName]
		if binding.MemoryLockLimits == nil {
			continue
		}

		if binding.MemoryLockLimits.Offset != nil {
			resOffset.Add(*binding.MemoryLockLimits.Offset)
		}
		if binding.MemoryLockLimits.LockGuestMemory {
			resRatio += count
		}
	}

	return resRatio, resOffset
}

func netBindingCountMap(
	vmi *v1.VirtualMachineInstance,
	registeredPlugins map[string]v1.InterfaceBindingPlugin,
) map[string]int64 {
	counter := map[string]int64{}
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.Binding == nil {
			continue
		}
		bindingName := iface.Binding.Name
		_, exists := registeredPlugins[bindingName]
		if !exists {
			continue
		}
		_, counterHasBinding := counter[bindingName]
		if counterHasBinding {
			counter[bindingName] += 1
		} else {
			counter[bindingName] = 1
		}
	}
	return counter
}
