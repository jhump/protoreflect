package protoprint

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/v2/internal"
)

func (p *Printer) printService(
	sd protoreflect.ServiceDescriptor,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	indent int,
) {
	si := sourceInfo.ByPath(path)
	p.printBlockElement(true, si, w, indent, func(w *writer, trailer func(int, bool)) {
		p.indent(w, indent)

		_, _ = fmt.Fprint(w, "service ")
		nameSi := sourceInfo.ByPath(append(path, internal.ServiceNameTag))
		p.printElementString(nameSi, w, indent, string(sd.Name()))
		_, _ = fmt.Fprintln(w, "{")
		indent++
		trailer(indent, true)

		opts := p.extractOptions(sd, reg, sd.Options())

		elements := elementAddrs{dsc: sd, opts: opts}
		elements.addrs = append(elements.addrs, optionsAsElementAddrs(internal.ServiceOptionsTag, -1, opts)...)
		methods := sd.Methods()
		for i, length := 0, methods.Len(); i < length; i++ {
			elements.addrs = append(elements.addrs, elementAddr{elementType: internal.ServiceMethodsTag, elementIndex: i})
		}

		p.sort(elements, sourceInfo, path)

		for i, el := range elements.addrs {
			if i > 0 {
				p.newLine(w)
			}

			childPath := append(path, el.elementType, int32(el.elementIndex))

			switch d := elements.at(el).(type) {
			case []option:
				p.printOptionsLong(d, reg, w, sourceInfo, childPath, indent)
			case protoreflect.MethodDescriptor:
				p.printMethod(d, reg, w, sourceInfo, childPath, indent)
			}
		}

		p.indent(w, indent-1)
		_, _ = fmt.Fprintln(w, "}")
	})
}

func (p *Printer) printMethod(
	mtd protoreflect.MethodDescriptor,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	indent int,
) {
	si := sourceInfo.ByPath(path)
	pkg := mtd.ParentFile().Package()
	p.printBlockElement(true, si, w, indent, func(w *writer, trailer func(int, bool)) {
		p.indent(w, indent)

		_, _ = fmt.Fprint(w, "rpc ")
		nameSi := sourceInfo.ByPath(append(path, internal.MethodNameTag))
		p.printElementString(nameSi, w, indent, string(mtd.Name()))

		_, _ = fmt.Fprint(w, "( ")
		inSi := sourceInfo.ByPath(append(path, internal.MethodInputTag))
		inName := p.qualifyTypeName(pkg, pkg, mtd.Input().FullName())
		if mtd.IsStreamingClient() {
			inName = "stream " + inName
		}
		p.printElementString(inSi, w, indent, inName)

		_, _ = fmt.Fprint(w, ") returns ( ")

		outSi := sourceInfo.ByPath(append(path, internal.MethodOutputTag))
		outName := p.qualifyTypeName(pkg, pkg, mtd.Output().FullName())
		if mtd.IsStreamingServer() {
			outName = "stream " + outName
		}
		p.printElementString(outSi, w, indent, outName)
		_, _ = fmt.Fprint(w, ") ")

		opts := p.extractOptions(mtd, reg, mtd.Options())

		if len(opts) > 0 {
			_, _ = fmt.Fprintln(w, "{")
			indent++
			trailer(indent, true)

			elements := elementAddrs{dsc: mtd, opts: opts}
			elements.addrs = optionsAsElementAddrs(internal.MethodOptionsTag, 0, opts)
			p.sort(elements, sourceInfo, path)

			for i, el := range elements.addrs {
				if i > 0 {
					p.newLine(w)
				}
				o := elements.at(el).([]option)
				childPath := append(path, el.elementType, int32(el.elementIndex))
				p.printOptionsLong(o, reg, w, sourceInfo, childPath, indent)
			}

			p.indent(w, indent-1)
			_, _ = fmt.Fprintln(w, "}")
		} else {
			_, _ = fmt.Fprint(w, ";")
			trailer(indent, false)
		}
	})
}
