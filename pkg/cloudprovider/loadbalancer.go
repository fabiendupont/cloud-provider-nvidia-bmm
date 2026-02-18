package cloudprovider

// LoadBalancer functionality is not currently supported by NVIDIA BMM.
// LoadBalancer services will need to use an external load balancer solution
// such as MetalLB, kube-vip, or a hardware load balancer.

// The LoadBalancer() method in nvidia_bmm_cloud.go returns cloudprovider.NotImplemented
// to indicate that this functionality is not available.

// Future implementation could integrate with:
// - External load balancer hardware at the site
// - Software load balancers like MetalLB
// - NVIDIA BMM-native load balancing if the platform adds support
