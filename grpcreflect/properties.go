package grpcreflect

import (
	"fmt"

	"google.golang.org/grpc"

	"github.com/jhump/protoreflect/desc"
)

// LoadServiceDescriptors loads the service descriptors for all services exposed by the
// given GRPC server.
func LoadServiceDescriptors(s *grpc.Server) (map[string]*desc.ServiceDescriptor, error) {
	descs := map[string]*desc.ServiceDescriptor{}
	for name, info := range s.GetServiceInfo() {
		file, ok := info.Metadata.(string)
		if !ok {
			return nil, fmt.Errorf("Service %q has unexpected metadata. Expecting a string, got %v", name, info.Metadata)
		}
		fd, err := desc.LoadFileDescriptor(file)
		if err != nil {
			return nil, err
		}
		d := fd.FindSymbol(name)
		if d == nil {
			return nil, fmt.Errorf("File descriptor for %q has no element named %q", file, name)
		}
		sd, ok := d.(*desc.ServiceDescriptor)
		if !ok {
			return nil, fmt.Errorf("File descriptor for %q has incorrect element named %q. Expecting a service descriptor, got %v", file, name, d)
		}
		descs[name] = sd
	}
	return descs, nil
}
