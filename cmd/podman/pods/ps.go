package pods

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/containers/podman/v2/cmd/podman/registry"
	"github.com/containers/podman/v2/cmd/podman/validate"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	psDescription = "List all pods on system including their names, ids and current state."

	// Command: podman pod _ps_
	psCmd = &cobra.Command{
		Use:     "ps",
		Aliases: []string{"ls", "list"},
		Short:   "List pods",
		Long:    psDescription,
		RunE:    pods,
		Args:    validate.NoArgs,
	}
)

var (
	defaultHeaders = "POD ID\tNAME\tSTATUS\tCREATED"
	inputFilters   []string
	noTrunc        bool
	psInput        entities.PodPSOptions
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: psCmd,
		Parent:  podCmd,
	})
	flags := psCmd.Flags()
	flags.BoolVar(&psInput.CtrNames, "ctr-names", false, "Display the container names")
	flags.BoolVar(&psInput.CtrIds, "ctr-ids", false, "Display the container UUIDs. If no-trunc is not set they will be truncated")
	flags.BoolVar(&psInput.CtrStatus, "ctr-status", false, "Display the container status")
	// TODO should we make this a [] ?
	flags.StringSliceVarP(&inputFilters, "filter", "f", []string{}, "Filter output based on conditions given")
	flags.StringVar(&psInput.Format, "format", "", "Pretty-print pods to JSON or using a Go template")
	flags.BoolVar(&psInput.Namespace, "namespace", false, "Display namespace information of the pod")
	flags.BoolVar(&psInput.Namespace, "ns", false, "Display namespace information of the pod")
	flags.BoolVar(&noTrunc, "no-trunc", false, "Do not truncate pod and container IDs")
	flags.BoolVarP(&psInput.Quiet, "quiet", "q", false, "Print the numeric IDs of the pods only")
	flags.StringVar(&psInput.Sort, "sort", "created", "Sort output by created, id, name, or number")
	validate.AddLatestFlag(psCmd, &psInput.Latest)
}

func pods(cmd *cobra.Command, _ []string) error {
	var (
		w   io.Writer = os.Stdout
		row string
	)

	if psInput.Quiet && len(psInput.Format) > 0 {
		return errors.New("quiet and format cannot be used together")
	}
	if cmd.Flag("filter").Changed {
		psInput.Filters = make(map[string][]string)
		for _, f := range inputFilters {
			split := strings.Split(f, "=")
			if len(split) < 2 {
				return errors.Errorf("filter input must be in the form of filter=value: %s is invalid", f)
			}
			psInput.Filters[split[0]] = append(psInput.Filters[split[0]], split[1])
		}
	}
	responses, err := registry.ContainerEngine().PodPs(context.Background(), psInput)
	if err != nil {
		return err
	}

	if err := sortPodPsOutput(psInput.Sort, responses); err != nil {
		return err
	}

	if psInput.Format == "json" {
		b, err := json.MarshalIndent(responses, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}

	lpr := make([]ListPodReporter, 0, len(responses))
	for _, r := range responses {
		lpr = append(lpr, ListPodReporter{r})
	}
	headers, row := createPodPsOut()
	if psInput.Quiet {
		row = "{{.Id}}\n"
	}
	if cmd.Flag("format").Changed {
		row = psInput.Format
		if !strings.HasPrefix(row, "\n") {
			row += "\n"
		}
	}
	format := "{{range . }}" + row + "{{end}}"
	if !psInput.Quiet && !cmd.Flag("format").Changed {
		format = headers + format
	}
	tmpl, err := template.New("listPods").Parse(format)
	if err != nil {
		return err
	}
	if !psInput.Quiet {
		w = tabwriter.NewWriter(os.Stdout, 8, 2, 2, ' ', 0)
	}
	if err := tmpl.Execute(w, lpr); err != nil {
		return err
	}
	if flusher, ok := w.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

func createPodPsOut() (string, string) {
	var row string
	headers := defaultHeaders
	row += "{{.Id}}"

	row += "\t{{.Name}}\t{{.Status}}\t{{.Created}}"

	if psInput.CtrIds {
		headers += "\tIDS"
		row += "\t{{.ContainerIds}}"
	}
	if psInput.CtrNames {
		headers += "\tNAMES"
		row += "\t{{.ContainerNames}}"
	}
	if psInput.CtrStatus {
		headers += "\tSTATUS"
		row += "\t{{.ContainerStatuses}}"
	}
	if psInput.Namespace {
		headers += "\tCGROUP\tNAMESPACES"
		row += "\t{{.Cgroup}}\t{{.Namespace}}"
	}
	if !psInput.CtrStatus && !psInput.CtrNames && !psInput.CtrIds {
		headers += "\t# OF CONTAINERS"
		row += "\t{{.NumberOfContainers}}"

	}
	headers += "\tINFRA ID\n"
	row += "\t{{.InfraId}}\n"
	return headers, row
}

// ListPodReporter is a struct for pod ps output
type ListPodReporter struct {
	*entities.ListPodsReport
}

// Created returns a human readable created time/date
func (l ListPodReporter) Created() string {
	return units.HumanDuration(time.Since(l.ListPodsReport.Created)) + " ago"
}

// Labels returns a map of the pod's labels
func (l ListPodReporter) Labels() map[string]string {
	return l.ListPodsReport.Labels
}

// NumberofContainers returns an int representation for
// the number of containers belonging to the pod
func (l ListPodReporter) NumberOfContainers() int {
	return len(l.Containers)
}

// ID is a wrapper to Id for compat, typos
func (l ListPodReporter) ID() string {
	return l.Id()
}

// Id returns the Pod id
func (l ListPodReporter) Id() string { //nolint
	if noTrunc {
		return l.ListPodsReport.Id
	}
	return l.ListPodsReport.Id[0:12]
}

// Added for backwards compatibility with podmanv1
func (l ListPodReporter) InfraID() string {
	return l.InfraId()
}

// InfraId returns the infra container id for the pod
// depending on trunc
func (l ListPodReporter) InfraId() string { //nolint
	if len(l.ListPodsReport.InfraId) == 0 {
		return ""
	}
	if noTrunc {
		return l.ListPodsReport.InfraId
	}
	return l.ListPodsReport.InfraId[0:12]
}

func (l ListPodReporter) ContainerIds() string {
	ctrids := make([]string, 0, len(l.Containers))
	for _, c := range l.Containers {
		id := c.Id
		if !noTrunc {
			id = id[0:12]
		}
		ctrids = append(ctrids, id)
	}
	return strings.Join(ctrids, ",")
}

func (l ListPodReporter) ContainerNames() string {
	ctrNames := make([]string, 0, len(l.Containers))
	for _, c := range l.Containers {
		ctrNames = append(ctrNames, c.Names)
	}
	return strings.Join(ctrNames, ",")
}

func (l ListPodReporter) ContainerStatuses() string {
	statuses := make([]string, 0, len(l.Containers))
	for _, c := range l.Containers {
		statuses = append(statuses, c.Status)
	}
	return strings.Join(statuses, ",")
}

func sortPodPsOutput(sortBy string, lprs []*entities.ListPodsReport) error {
	switch sortBy {
	case "created":
		sort.Sort(podPsSortedCreated{lprs})
	case "id":
		sort.Sort(podPsSortedID{lprs})
	case "name":
		sort.Sort(podPsSortedName{lprs})
	case "number":
		sort.Sort(podPsSortedNumber{lprs})
	case "status":
		sort.Sort(podPsSortedStatus{lprs})
	default:
		return errors.Errorf("invalid option for --sort, options are: id, names, or number")
	}
	return nil
}

type lprSort []*entities.ListPodsReport

func (a lprSort) Len() int      { return len(a) }
func (a lprSort) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type podPsSortedCreated struct{ lprSort }

func (a podPsSortedCreated) Less(i, j int) bool {
	return a.lprSort[i].Created.After(a.lprSort[j].Created)
}

type podPsSortedID struct{ lprSort }

func (a podPsSortedID) Less(i, j int) bool { return a.lprSort[i].Id < a.lprSort[j].Id }

type podPsSortedNumber struct{ lprSort }

func (a podPsSortedNumber) Less(i, j int) bool {
	return len(a.lprSort[i].Containers) < len(a.lprSort[j].Containers)
}

type podPsSortedName struct{ lprSort }

func (a podPsSortedName) Less(i, j int) bool { return a.lprSort[i].Name < a.lprSort[j].Name }

type podPsSortedStatus struct{ lprSort }

func (a podPsSortedStatus) Less(i, j int) bool {
	return a.lprSort[i].Status < a.lprSort[j].Status
}
