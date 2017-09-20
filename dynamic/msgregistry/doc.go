// Package msgregistry contains a registry of known message and enum types.
// The MessageRegistry is used for interacting with Any messages where the
// actual embedded value may be a dynamic message. There is also functionality
// for resolving type URLs into descriptors, which supports dynamically loading
// type descriptions (represented using the well-known types: Api, Method, Type,
// Field, Enum, and EnumValue) and converting them to descriptors. This allows
// for using these dynamically loaded schemas using dynamic messages. The
// registry also exposes related functionality for inter-op between descriptors
// and the proto well-known types that model APIs and types.
package msgregistry
