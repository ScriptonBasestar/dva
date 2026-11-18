package cirun

const maxHistoryStepObservations = 10000

type HistoryObservation struct {
	RunID        string                   `json:"run_id"`
	Admission    *AdmissionObservation    `json:"admission"`
	NativeStages *NativeStageObservation  `json:"native_stages,omitempty"`
	Steps        []HistoryStepObservation `json:"steps"`
	OmittedSteps int                      `json:"omitted_steps"`
}

type HistoryStepObservation struct {
	Name              string                `json:"name"`
	Status            string                `json:"status"`
	Admission         *AdmissionObservation `json:"admission"`
	OutputObservation *OutputObservation    `json:"output_observation"`
	Workdir           string                `json:"workdir,omitempty"`
	WorkerBudget      *WorkerBudget         `json:"worker_budget,omitempty"`
	OSWaitResources   *OSWaitResources      `json:"os_wait_resources,omitempty"`
}

func projectHistoryObservation(r Report, remaining *int) HistoryObservation {
	o := HistoryObservation{RunID: r.ID, Admission: r.Admission, NativeStages: r.NativeStages, Steps: []HistoryStepObservation{}}
	for _, s := range r.Steps {
		if *remaining == 0 {
			o.OmittedSteps++
			continue
		}
		o.Steps = append(o.Steps, HistoryStepObservation{Name: s.Name, Status: s.Status, Admission: s.Admission, OutputObservation: s.OutputObservation, Workdir: s.Workdir, WorkerBudget: s.WorkerBudget, OSWaitResources: s.OSWaitResources})
		*remaining--
	}
	return o
}
