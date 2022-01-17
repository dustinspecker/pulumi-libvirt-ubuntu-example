package main

import (
	"os"

	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		conf := config.New(ctx, "")

		// require each stack to specify a libvirt_uri
		libvirt_uri := conf.Require("libvirt_uri")
		// create a provider, this isn't required, but will make it easier to configure
		// a libvirt_uri, which we'll discuss in a bit
		provider, err := libvirt.NewProvider(ctx, "provider", &libvirt.ProviderArgs{
			Uri: pulumi.String(libvirt_uri),
		})
		if err != nil {
			return err
		}

		// `pool` is a storage pool that can be used to create volumes
		// the `dir` type uses a directory to manage files
		// `Path` maps to a directory on the host filesystem, so we'll be able to
		// volume contents in `/pool/cluster_storage/`
		pool, err := libvirt.NewPool(ctx, "cluster", &libvirt.PoolArgs{
			Type: pulumi.String("dir"),
			Path: pulumi.String("/pool/cluster_storage"),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		// create a volume with the contents being a Ubuntu 20.04 server image
		ubuntu, err := libvirt.NewVolume(ctx, "ubuntu", &libvirt.VolumeArgs{
			Pool:   pool.Name,
			Source: pulumi.String("https://cloud-images.ubuntu.com/releases/focal/release/ubuntu-20.04-server-cloudimg-amd64.img"),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		// create a filesystem volume for our VM
		// This filesystem will be based on the `ubuntu` volume above
		// we'll use a size of 10GB
		filesystem, err := libvirt.NewVolume(ctx, "filesystem", &libvirt.VolumeArgs{
			BaseVolumeId: ubuntu.ID(),
			Pool:         pool.Name,
			Size:         pulumi.Int(10000000000),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		cloud_init_user_data, err := os.ReadFile("./cloud_init_user_data.yaml")
		if err != nil {
			return err
		}

		cloud_init_network_config, err := os.ReadFile("./cloud_init_network_config.yaml")
		if err != nil {
			return err
		}

		// create a cloud init disk that will setup the ubuntu credentials
		cloud_init, err := libvirt.NewCloudInitDisk(ctx, "cloud-init", &libvirt.CloudInitDiskArgs{
			MetaData:      pulumi.String(string(cloud_init_user_data)),
			NetworkConfig: pulumi.String(string(cloud_init_network_config)),
			Pool:          pool.Name,
			UserData:      pulumi.String(string(cloud_init_user_data)),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		// create NAT network using 192.168.10/24 CIDR
		network, err := libvirt.NewNetwork(ctx, "network", &libvirt.NetworkArgs{
			Addresses: pulumi.StringArray{pulumi.String("192.168.10.0/24")},
			Autostart: pulumi.Bool(true),
			Mode:      pulumi.String("nat"),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		// create a VM that has a name starting with ubuntu
		domain, err := libvirt.NewDomain(ctx, "ubuntu", &libvirt.DomainArgs{
			Autostart: pulumi.Bool(true),
			Cloudinit: cloud_init.ID(),
			Consoles: libvirt.DomainConsoleArray{
				// enables using `virsh console ...`
				libvirt.DomainConsoleArgs{
					Type:       pulumi.String("pty"),
					TargetPort: pulumi.String("0"),
					TargetType: pulumi.String("serial"),
				},
			},
			Disks: libvirt.DomainDiskArray{
				libvirt.DomainDiskArgs{
					VolumeId: filesystem.ID(),
				},
			},
			NetworkInterfaces: libvirt.DomainNetworkInterfaceArray{
				libvirt.DomainNetworkInterfaceArgs{
					NetworkId:    network.ID(),
					WaitForLease: pulumi.Bool(true),
				},
			},
			// delete existing VM before creating replacement to avoid two VMs trying to use the same volume
		}, pulumi.Provider(provider), pulumi.ReplaceOnChanges([]string{"*"}), pulumi.DeleteBeforeReplace(true))

		ctx.Export("IP Address", domain.NetworkInterfaces.Index(pulumi.Int(0)).Addresses().Index(pulumi.Int(0)))
		ctx.Export("VM name", domain.Name)

		return nil
	})
}
