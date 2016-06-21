export interface SlackSettings {
  webhookURL: string;
}

export interface SlackModel {
  enabled: boolean;
  settings: SlackSettings;
}
