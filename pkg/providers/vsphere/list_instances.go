package vsphere

import (
	"github.com/Sirupsen/logrus"
	"github.com/emc-advanced-dev/unik/pkg/providers/common"
	"github.com/emc-advanced-dev/unik/pkg/types"
	unikutil "github.com/emc-advanced-dev/unik/pkg/util"
	"github.com/layer-x/layerx-commons/lxerrors"
	vspheretypes "github.com/vmware/govmomi/vim25/types"
	"time"
)

func (p *VsphereProvider) ListInstances() ([]*types.Instance, error) {
	c := p.getClient()
	vms, err := c.Vms()
	if err != nil {
		return nil, lxerrors.New("getting vsphere vms", err)
	}

	instances := []*types.Instance{}
	for _, vm := range vms {
		if vm.Config == nil {
			continue
		}
		//we use mac address as the vm id
		macAddr := ""
		if vm.Config != nil && vm.Config.Hardware.Device != nil {
		FindEthLoop:
			for _, device := range vm.Config.Hardware.Device {
				switch device.(type) {
				case *vspheretypes.VirtualE1000:
					eth := device.(*vspheretypes.VirtualE1000)
					macAddr = eth.MacAddress
					break FindEthLoop
				case *vspheretypes.VirtualE1000e:
					eth := device.(*vspheretypes.VirtualE1000e)
					macAddr = eth.MacAddress
					break FindEthLoop
				case *vspheretypes.VirtualPCNet32:
					eth := device.(*vspheretypes.VirtualPCNet32)
					macAddr = eth.MacAddress
					break FindEthLoop
				case *vspheretypes.VirtualSriovEthernetCard:
					eth := device.(*vspheretypes.VirtualSriovEthernetCard)
					macAddr = eth.MacAddress
					break FindEthLoop
				case *vspheretypes.VirtualVmxnet:
					eth := device.(*vspheretypes.VirtualVmxnet)
					macAddr = eth.MacAddress
					break FindEthLoop
				case *vspheretypes.VirtualVmxnet2:
					eth := device.(*vspheretypes.VirtualVmxnet2)
					macAddr = eth.MacAddress
					break FindEthLoop
				case *vspheretypes.VirtualVmxnet3:
					eth := device.(*vspheretypes.VirtualVmxnet3)
					macAddr = eth.MacAddress
					break FindEthLoop
				}
			}
		}
		if macAddr == "" {
			logrus.WithFields(logrus.Fields{"vm": vm}).Warnf("vm found, cannot identify mac addr")
			continue
		}
		instance, ok := p.state.GetInstances()[instanceId]
		if !ok {
			logrus.WithFields(logrus.Fields{"vm": vm, "instance-id": vm.Config.InstanceUuid}).Warnf("vm found, cannot identify instance id")
			continue
		}

		switch vm.Summary.Runtime.PowerState {
		case "poweredOn":
			instance.State = types.InstanceState_Running
			break
		case "poweredOff":
		case "suspended":
			instance.State = types.InstanceState_Stopped
			break
		default:
			instance.State = types.InstanceState_Unknown
			break
		}

		instanceListenerIp, err := c.GetVmIp(VsphereInstanceListener)
		if err != nil {
			return nil, lxerrors.New("failed to retrieve instance listener ip. is unik instance listener running?", err)
		}

		if err := unikutil.Retry(5, time.Duration(2000*time.Millisecond), func() error {
			logrus.Debugf("getting instance ip")
			instance.IpAddress, err = common.GetInstanceIp(instanceListenerIp, 3000, macAddr)
			if err != nil {
				return err
			}
			return nil
		}); err != nil {
			return nil, lxerrors.New("failed to retrieve instance ip", err)
		}

		err = p.state.ModifyInstances(func(instances map[string]*types.Instance) error {
			instances[instance.Id] = instance
			return nil
		})
		if err != nil {
			return nil, lxerrors.New("saving instance to state", err)
		}

		instances = append(instances, instance)
	}
	return instances, nil
}
