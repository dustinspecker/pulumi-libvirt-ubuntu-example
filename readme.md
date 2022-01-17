# pulumi-libvirt-ubuntu-example

> Based on https://dustinspecker.com/posts/ubuntu-vm-pulumi-libvirt/

## Usage

1. Install [Pulumi](https://www.pulumi.com/)
1. Clone this repository
1. Run `pulumi login`
1. Run `pulumi stack init dev`
1. Run `pulumi config set libvirt_uri qemu:///system`
1. Run `pulumi up`
