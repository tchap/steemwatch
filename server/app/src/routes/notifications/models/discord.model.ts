export interface DiscordSettings {
  startToken: string;
  username: string;
}

export interface DiscordModel {
  enabled: boolean;
  settings: DiscordSettings;
}
