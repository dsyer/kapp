package tools

import (
	"fmt"

	"github.com/cppforlife/go-cli-ui/ui"
	uitable "github.com/cppforlife/go-cli-ui/ui/table"
	cmdcore "github.com/k14s/kapp/pkg/kapp/cmd/core"
	"github.com/k14s/kapp/pkg/kapp/logger"
	ctlres "github.com/k14s/kapp/pkg/kapp/resources"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/labels"
)

type ListLabelsOptions struct {
	ui          ui.UI
	depsFactory cmdcore.DepsFactory
	logger      logger.Logger

	FileFlags FileFlags
	Accessor  string
	Values    bool
}

func NewListLabelsOptions(ui ui.UI, depsFactory cmdcore.DepsFactory, logger logger.Logger) *ListLabelsOptions {
	return &ListLabelsOptions{ui: ui, depsFactory: depsFactory, logger: logger}
}

func NewListLabelsCmd(o *ListLabelsOptions, flagsFactory cmdcore.FlagsFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list-labels",
		Aliases: []string{"ls-labels"},
		Short:   "List labels",
		RunE:    func(_ *cobra.Command, _ []string) error { return o.Run() },
	}
	o.FileFlags.Set(cmd)
	cmd.Flags().StringVar(&o.Accessor, "accessor", "labels", "Extracted field (labels, annotations, ownerrefs)")
	cmd.Flags().BoolVar(&o.Values, "values", false, "Show values")
	return cmd
}

func (o *ListLabelsOptions) Run() error {
	rs, err := o.listResources()
	if err != nil {
		return err
	}

	data := map[string]map[string]int{}

	resAccessor, ok := resAccessors[o.Accessor]
	if !ok {
		return fmt.Errorf("Unknown resource accessor")
	}

	for _, res := range rs {
		kvs := resAccessor.KVs(res)
		for k, v := range kvs {
			if _, found := data[k]; !found {
				data[k] = map[string]int{}
			}
			data[k][v]++
		}
		if len(kvs) == 0 {
			data[""] = map[string]int{}
		}
	}

	valueHeader := uitable.NewHeader("Value")
	valueHeader.Hidden = !o.Values

	table := uitable.Table{
		Title:   "Labels",
		Content: "labels",

		Header: []uitable.Header{
			uitable.NewHeader("Name"),
			valueHeader,
			uitable.NewHeader("Resources"),
		},

		SortBy: []uitable.ColumnSort{
			{Column: 0, Asc: true},
			{Column: 1, Asc: true},
		},
	}

	for name, counts := range data {
		totalCount := 0

		for val, count := range counts {
			totalCount += count

			if o.Values {
				table.Rows = append(table.Rows, []uitable.Value{
					uitable.NewValueString(name),
					uitable.NewValueString(val),
					uitable.NewValueInt(count),
				})
			}
		}

		if !o.Values {
			table.Rows = append(table.Rows, []uitable.Value{
				uitable.NewValueString(name),
				uitable.NewValueString(""),
				uitable.NewValueInt(totalCount),
			})
		}
	}

	o.ui.PrintTable(table)

	return nil
}

func (o *ListLabelsOptions) listResources() ([]ctlres.Resource, error) {
	coreClient, err := o.depsFactory.CoreClient()
	if err != nil {
		return nil, err
	}

	dynamicClient, err := o.depsFactory.DynamicClient()
	if err != nil {
		return nil, err
	}

	identifiedResources := ctlres.NewIdentifiedResources(coreClient, dynamicClient, nil, o.logger)

	labelSelector, err := labels.Parse("!kapp")
	if err != nil {
		return nil, err
	}

	return ctlres.NewLabeledResources(labelSelector, identifiedResources, o.logger).All()
}

type resourceAccessor struct {
	KVs func(ctlres.Resource) map[string]string
}

var (
	resAccessors = map[string]resourceAccessor{
		"labels": resourceAccessor{
			KVs: func(res ctlres.Resource) map[string]string { return res.Labels() },
		},
		"annotations": resourceAccessor{
			KVs: func(res ctlres.Resource) map[string]string { return res.Annotations() },
		},
		"ownerrefs": resourceAccessor{
			KVs: func(res ctlres.Resource) map[string]string {
				result := map[string]string{}
				for _, ref := range res.OwnerRefs() {
					result[res.Namespace()+"/"+ref.APIVersion+"/"+ref.Kind+"/"+ref.Name] = ""
				}
				return result
			},
		},
	}
)
