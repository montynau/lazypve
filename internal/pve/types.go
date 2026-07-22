package pve

// Node is a Proxmox VE cluster node, as returned by GET /nodes.
type Node struct {
	Node    string  `json:"node"`
	Status  string  `json:"status"`
	CPU     float64 `json:"cpu"`
	MaxCPU  int     `json:"maxcpu"`
	Mem     int64   `json:"mem"`
	MaxMem  int64   `json:"maxmem"`
	Disk    int64   `json:"disk"`
	MaxDisk int64   `json:"maxdisk"`
	Uptime  int64   `json:"uptime"`
}

// VM is a QEMU guest, as returned by GET /nodes/{node}/qemu.
type VM struct {
	VMID    int     `json:"vmid"`
	Name    string  `json:"name"`
	Status  string  `json:"status"`
	CPU     float64 `json:"cpu"`
	MaxCPU  int     `json:"cpus"`
	Mem     int64   `json:"mem"`
	MaxMem  int64   `json:"maxmem"`
	Disk    int64   `json:"disk"`
	MaxDisk int64   `json:"maxdisk"`
	NetIn   int64   `json:"netin"`
	NetOut  int64   `json:"netout"`
	Uptime  int64   `json:"uptime"`
}

// Container is an LXC guest, as returned by GET /nodes/{node}/lxc.
type Container struct {
	VMID    int     `json:"vmid"`
	Name    string  `json:"name"`
	Status  string  `json:"status"`
	CPU     float64 `json:"cpu"`
	MaxCPU  int     `json:"cpus"`
	Mem     int64   `json:"mem"`
	MaxMem  int64   `json:"maxmem"`
	Disk    int64   `json:"disk"`
	MaxDisk int64   `json:"maxdisk"`
	NetIn   int64   `json:"netin"`
	NetOut  int64   `json:"netout"`
	Uptime  int64   `json:"uptime"`
}

type response[T any] struct {
	Data T `json:"data"`
}
