export interface Field {
  id:          string;
  label:       string;
  description: string;
}

export interface Event {
  id:          string;
  title:       string;
  description: string;
  fields:      Field[];
}

export interface Notifier {
  id:           string;
  title:        string;
  titleIconURL: string;
  fields:       Field[];
}
