import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface User {
  UserID: string;
  Email: string;
  DisplayName: string;
}

export interface AuthResponse {
  user: User;
  token: string;
}

export interface Pool {
  PoolID: string;
  Name: string;
}

export interface Invitation {
  InvitationID: string;
  Token: string;
  ExpiresAt: string;
}

export interface Prediction {
  PredictionID: string;
  HomeGoals: number;
  AwayGoals: number;
}

export interface BracketResult {
  BracketID: string;
  UpdatedAt: string;
}

export interface MatchItem {
  id: string;
  stage: string;
  group_id?: string;
  home_team_id: string;
  away_team_id: string;
  kickoff_at: string;
  venue: string;
  status: string;
  home_goals?: number;
  away_goals?: number;
}

@Injectable({ providedIn: 'root' })
export class ApiService {
  private baseUrl = 'http://localhost:8080/api/v1';

  constructor(private http: HttpClient) {}

  // Auth
  register(email: string, password: string, displayName: string): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${this.baseUrl}/auth/register`, {
      email, password, display_name: displayName
    });
  }

  login(email: string, password: string): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${this.baseUrl}/auth/login`, { email, password });
  }

  // Pools
  createPool(name: string, description: string, tournamentId: string): Observable<{ pool: Pool }> {
    return this.http.post<{ pool: Pool }>(`${this.baseUrl}/pools`, {
      name, description, tournament_id: tournamentId
    });
  }

  getPools(): Observable<{ pools: Pool[] }> {
    return this.http.get<{ pools: Pool[] }>(`${this.baseUrl}/pools`);
  }

  inviteMember(poolId: string, email: string): Observable<{ invitation: Invitation }> {
    return this.http.post<{ invitation: Invitation }>(
      `${this.baseUrl}/pools/${poolId}/invitations`, { email });
  }

  acceptInvitation(token: string): Observable<{ accepted: { PoolID: string } }> {
    return this.http.post<{ accepted: { PoolID: string } }>(
      `${this.baseUrl}/invitations/${token}/accept`, {});
  }

  // Match predictions
  submitPrediction(poolId: string, matchId: string, homeGoals: number, awayGoals: number): Observable<{ prediction: Prediction }> {
    return this.http.post<{ prediction: Prediction }>(
      `${this.baseUrl}/pools/${poolId}/predictions`, {
        match_id: matchId, home_goals: homeGoals, away_goals: awayGoals
      });
  }

  // Bracket
  submitBracket(poolId: string, data: BracketPayload): Observable<{ bracket: BracketResult }> {
    return this.http.post<{ bracket: BracketResult }>(
      `${this.baseUrl}/pools/${poolId}/bracket`, data);
  }

  // Matches
  getMatches(tournamentId: string): Observable<{ matches: MatchItem[] }> {
    return this.http.get<{ matches: MatchItem[] }>(
      `${this.baseUrl}/matches?tournament_id=${tournamentId}`);
  }

  // Teams
  getTeams(tournamentId: string): Observable<{ teams: TeamInfo[] }> {
    return this.http.get<{ teams: TeamInfo[] }>(`${this.baseUrl}/teams?tournament_id=${tournamentId}`);
  }

  // Scores / Ranking
  getRanking(poolId: string): Observable<{ ranking: RankingEntry[] }> {
    return this.http.get<{ ranking: RankingEntry[] }>(
      `${this.baseUrl}/pools/${poolId}/ranking`);
  }
}

export interface BracketPayload {
  tournament_id: string;
  teams_to_round_of_32: string[];
  teams_to_round_of_16: string[];
  teams_to_quarter_final: string[];
  teams_to_semi_final: string[];
  teams_to_final: string[];
  champion: string;
  third_place_winner: string;
}

export interface RankingEntry {
  user_id: string;
  display_name: string;
  total_points: number;
}

export interface TeamInfo {
  id: string;
  code: string;
  name: string;
}
