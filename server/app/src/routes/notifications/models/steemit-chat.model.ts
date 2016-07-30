export interface SteemitChatSettings {
  username:  string;
  userID:    string;
  authToken: string;
}

export interface SteemitChatModel {
  settings: SteemitChatSettings;
  enabled:  boolean;
}
