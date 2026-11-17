package cirun

const maxHistoryStepObservations = 10000

type HistoryObservation struct {
	RunID        string                   `json:"run_id"`
	Admission    *AdmissionObservation    `json:"admission"`
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
}

func projectHistoryObservation(r Report, remaining *int) HistoryObservation {
	o := HistoryObservation{RunID: r.ID, Admission: r.Admission, Steps: []HistoryStepObservation{}}
	for _, s := range r.Steps {
		if *remaining == 0 {
			o.OmittedSteps++
			continue
		}
		o.Steps = append(o.Steps, HistoryStepObservation{Name: s.Name, Status: s.Status, Admission: s.Admission, OutputObservation: s.OutputObservation, Workdir: s.Workdir, WorkerBudget: s.WorkerBudget})
		*remaining--
	}
	return o
}
