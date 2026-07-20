import { Metadata } from './common';

/** Organization (spec: OrgResponse). */
export interface OrgResponse {
  country: string;
  created_at: string;
  id: string;
  metadata: Metadata;
  name: string;
  status: string;
  timezone: string;
  updated_at: string;
}

/** Create organization input (spec: CreateOrgRequest). */
export interface CreateOrgRequest {
  country: string;
  metadata?: Metadata;
  name: string;
  timezone: string;
}
