package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourcesConfig) DeepCopyInto(out *ResourcesConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is a deepcopy function, copying the receiver, creating a new ResourcesConfig.
func (in *ResourcesConfig) DeepCopy() *ResourcesConfig {
	if in == nil {
		return nil
	}
	out := new(ResourcesConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is a deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ResourcesConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourcesConfigList) DeepCopyInto(out *ResourcesConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ResourcesConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is a deepcopy function, copying the receiver, creating a new ResourcesConfigList.
func (in *ResourcesConfigList) DeepCopy() *ResourcesConfigList {
	if in == nil {
		return nil
	}
	out := new(ResourcesConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is a deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ResourcesConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourcesConfigSpec) DeepCopyInto(out *ResourcesConfigSpec) {
	*out = *in
	out.TargetRef = in.TargetRef
	if in.ContainerConfigs != nil {
		in, out := &in.ContainerConfigs, &out.ContainerConfigs
		*out = make(map[string]ContainerConfig, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is a deepcopy function, copying the receiver, creating a new ResourcesConfigSpec.
func (in *ResourcesConfigSpec) DeepCopy() *ResourcesConfigSpec {
	if in == nil {
		return nil
	}
	out := new(ResourcesConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourcesConfigStatus) DeepCopyInto(out *ResourcesConfigStatus) {
	*out = *in
	if !in.LastUpdateTime.IsZero() {
		out.LastUpdateTime = in.LastUpdateTime
	}
	if !in.LastSyncTime.IsZero() {
		out.LastSyncTime = in.LastSyncTime
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is a deepcopy function, copying the receiver, creating a new ResourcesConfigStatus.
func (in *ResourcesConfigStatus) DeepCopy() *ResourcesConfigStatus {
	if in == nil {
		return nil
	}
	out := new(ResourcesConfigStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetRef) DeepCopyInto(out *TargetRef) {
	*out = *in
}

// DeepCopy is a deepcopy function, copying the receiver, creating a new TargetRef.
func (in *TargetRef) DeepCopy() *TargetRef {
	if in == nil {
		return nil
	}
	out := new(TargetRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerConfig) DeepCopyInto(out *ContainerConfig) {
	*out = *in
}

// DeepCopy is a deepcopy function, copying the receiver, creating a new ContainerConfig.
func (in *ContainerConfig) DeepCopy() *ContainerConfig {
	if in == nil {
		return nil
	}
	out := new(ContainerConfig)
	in.DeepCopyInto(out)
	return out
}
