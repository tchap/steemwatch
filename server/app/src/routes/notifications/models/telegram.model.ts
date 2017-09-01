export interface TelegramSettings {
  startToken: string;
  firstName: string;
  lastName: string;
  username: string;
}

export interface TelegramModel {
  enabled: boolean;
  settings: TelegramSettings;
}
