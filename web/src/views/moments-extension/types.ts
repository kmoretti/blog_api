export type ExtensionType = 'github' | 'website' | 'location' | 'music' | 'tweet'

export interface GithubPayload {
  repo_url: string
}

export interface WebsitePayload {
  title: string
  site: string
}

export interface LocationPayload {
  placeholder: string
  latitude: number
  longitude: number
}

export interface MusicPayload {
  url: string
}

export interface TweetPayload {
  url: string
  username: string
  status_id: string
}

export type ExtensionPayload =
  | GithubPayload
  | WebsitePayload
  | LocationPayload
  | MusicPayload
  | TweetPayload

export interface MomentExtension {
  type: ExtensionType
  payload: ExtensionPayload
}

export function parseExtension(json: string | undefined | null): MomentExtension | null {
  if (!json) return null
  try {
    const ext = JSON.parse(json) as MomentExtension
    if (ext && ext.type && ext.payload) return ext
    return null
  } catch {
    return null
  }
}
