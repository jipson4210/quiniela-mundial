import { Component, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { DatePipe, SlicePipe } from '@angular/common';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { ApiService, MatchItem } from '../../services/api.service';
import { ToastService } from '../../services/toast.service';
import { NavbarComponent } from '../../layout/navbar.component';
import { ToastComponent } from '../../layout/toast.component';
import { BracketViewComponent } from './bracket.component';

interface TeamInfo { id: string; code: string; name: string; }

type Tab = 'matches' | 'bracket' | 'ranking';

@Component({
  selector: 'app-pool',
  standalone: true,
  imports: [FormsModule, RouterModule, NavbarComponent, ToastComponent, BracketViewComponent, DatePipe, SlicePipe],
  template: `
    <app-navbar />
    <app-toast />
    <div class="container">
      <a routerLink="/dashboard" class="back">← Mis Grupos</a>
      <h1>{{ poolName || 'Quiniela' }}</h1>

      <div class="tabs">
        @for (t of tabs; track t) {
          <button [class.active]="activeTab === t" (click)="setTab(t)">{{ tabNames[t] }}</button>
        }
      </div>

      @if (activeTab === 'matches') {
        @if (loadingMatches) {
          <div class="skeleton-list">
            @for (i of [1,2,3,4,5]; track i) { <div class="skel-row"></div> }
          </div>
        } @else {
          @for (m of matches; track m.id) {
            <div class="match-card" [class.done]="m.status === 'finished'">
              <div class="match-header">
                <span class="stage-badge">{{ stageLabel(m.stage) }}</span>
                @if (m.group_id) { <span class="group-badge">G{{ getGroupLetter(m.group_id) }}</span> }
                <span class="kickoff">{{ m.kickoff_at | date:'dd/MM HH:mm' }}</span>
                <span class="venue">{{ m.venue }}</span>
                @if (m.status === 'finished') { <span class="done-badge">Finalizado</span> }
                @if (m.status === 'in_progress') { <span class="live-badge">🔴 En vivo</span> }
              </div>
              <div class="match-body">
                <div class="team home">
                  <span class="team-code">{{ teamMap[m.home_team_id]?.code || (m.home_team_id|slice:0:6) }}</span>
                </div>
                <div class="score-area">
                  <input class="goal-in" type="number" min="0" max="30" [(ngModel)]="preds[m.id].home"
                         [disabled]="m.status !== 'scheduled'" />
                  <span class="vs">vs</span>
                  <input class="goal-in" type="number" min="0" max="30" [(ngModel)]="preds[m.id].away"
                         [disabled]="m.status !== 'scheduled'" />
                </div>
                <div class="team away">
                  <span class="team-code">{{ teamMap[m.away_team_id]?.code || (m.away_team_id|slice:0:6) }}</span>
                </div>
                @if (m.status === 'scheduled') {
                  <button class="btn-save" (click)="savePred(m.id)">💾</button>
                }
                @if (m.status === 'finished' && m.home_goals !== undefined) {
                  <div class="result-badge">{{ m.home_goals }} - {{ m.away_goals }}</div>
                }
              </div>
            </div>
          }
        }
      }

      @if (activeTab === 'bracket') {
        <app-bracket-view [poolId]="poolId" [tournamentId]="bracketTid" />
      }

      @if (activeTab === 'ranking') {
        @if (loadingRanking) { <div class="skeleton-list">@for (i of [1,2,3]; track i) {<div class="skel-row"></div>}</div> }
        @else if (ranking.length === 0) {
          <div class="empty-state">🏆 <p>Aún no hay puntajes. ¡Sé el primero en predecir!</p></div>
        } @else {
          <div class="ranking-list">
            @for (r of ranking; track r.user_id; let i = $index) {
              <div class="rank-row" [class.top3]="i < 3">
                <span class="rank-pos">
                  @if (i === 0) {🥇} @else if (i === 1) {🥈} @else if (i === 2) {🥉} @else {#{{i+1}}}
                </span>
                <span class="rank-name">{{ r.display_name }}</span>
                <span class="rank-pts">{{ r.total_points }} pts</span>
                <div class="rank-bar" [style.width.%]="maxPts ? (r.total_points/maxPts*100) : 0"></div>
              </div>
            }
          </div>
        }
      }
    </div>

    <style>
      .container { max-width: 900px; margin: 0 auto; padding: 1.5rem 1rem; }
      .back { color: var(--color-primary); text-decoration: none; font-size: 0.9rem; display: inline-block; margin-bottom: 0.3rem; }
      h1 { color: var(--color-text); margin-bottom: 1rem; font-size: 1.5rem; }
      .tabs { display: flex; margin-bottom: 1.2rem; }
      .tabs button {
        padding: 0.55rem 1.4rem; border: 1px solid var(--color-border);
        background: var(--color-surface); color: var(--color-text); cursor: pointer; font-size: 0.9rem; transition: all 0.2s;
      }
      .tabs button:first-child { border-radius: 10px 0 0 10px; }
      .tabs button:last-child { border-radius: 0 10px 10px 0; }
      .tabs button.active { background: var(--color-primary); color: #fff; border-color: var(--color-primary); font-weight: 600; }

      .match-card {
        background: var(--color-surface); border-radius: 12px; padding: 0.9rem 1.2rem;
        margin-bottom: 0.5rem; box-shadow: 0 1px 6px rgba(0,0,0,0.04); transition: opacity 0.3s;
      }
      .match-card.done { opacity: 0.55; background: var(--color-bg); }
      .match-header { display: flex; gap: 0.6rem; align-items: center; margin-bottom: 0.5rem; font-size: 0.78rem; color: var(--color-text-secondary); flex-wrap: wrap; }
      .stage-badge { background: var(--color-primary); color: #fff; padding: 0.15rem 0.5rem; border-radius: 4px; font-weight: 600; }
      .group-badge { background: var(--color-accent); color: #fff; padding: 0.15rem 0.5rem; border-radius: 4px; }
      .done-badge { background: var(--color-text-secondary); color: #fff; padding: 0.15rem 0.5rem; border-radius: 4px; }
      .live-badge { animation: pulse 1.5s infinite; }
      .match-body { display: flex; align-items: center; gap: 0.6rem; }
      .team { flex: 1; text-align: center; }
      .team.home { text-align: right; }
      .team.away { text-align: left; }
      .team-code { font-weight: 700; font-size: 1rem; color: var(--color-text); }
      .score-area { display: flex; align-items: center; gap: 0.4rem; }
      .goal-in {
        width: 48px; height: 40px; text-align: center; font-size: 1.1rem; font-weight: 700;
        border: 2px solid var(--color-border); border-radius: 10px;
        background: var(--color-bg); color: var(--color-text);
      }
      .goal-in:focus { border-color: var(--color-primary); outline: none; }
      .goal-in:disabled { opacity: 0.5; cursor: not-allowed; }
      .vs { font-weight: 700; font-size: 0.85rem; color: var(--color-text-secondary); margin: 0 0.3rem; }
      .btn-save {
        background: var(--color-primary); color: #fff; border: none;
        padding: 0.5rem 0.8rem; border-radius: 8px; cursor: pointer; font-size: 0.9rem; transition: transform 0.15s;
      }
      .btn-save:hover { transform: scale(1.05); }
      .btn-save:disabled { opacity: 0.4; cursor: not-allowed; }
      .result-badge { background: var(--color-accent); color: #fff; padding: 0.3rem 0.7rem; border-radius: 8px; font-weight: 700; font-size: 1rem; }

      .card { background: var(--color-surface); border-radius: 12px; padding: 1.5rem; box-shadow: 0 2px 8px rgba(0,0,0,0.04); }
      .hint { font-size: 0.85rem; color: var(--color-text-secondary); margin-bottom: 1rem; line-height: 1.4; }
      .inp { width: 100%; padding: 0.55rem; border: 1px solid var(--color-border); border-radius: 8px; background: var(--color-bg); color: var(--color-text); font-size: 0.85rem; margin-bottom: 0.8rem; font-family: monospace; }
      textarea.inp { resize: vertical; }
      .btn { background: var(--color-primary); color: #fff; border: none; padding: 0.65rem 1.5rem; border-radius: 10px; font-weight: 600; cursor: pointer; transition: transform 0.15s; }
      .btn:hover { transform: scale(1.02); }

      .rank-row {
        display: flex; align-items: center; gap: 1rem; padding: 0.8rem 1rem;
        background: var(--color-surface); border-radius: 10px; margin-bottom: 0.4rem;
        box-shadow: 0 1px 4px rgba(0,0,0,0.03); position: relative; overflow: hidden;
      }
      .rank-row.top3 { border-left: 4px solid var(--color-accent); }
      .rank-pos { font-size: 1.2rem; min-width: 2rem; text-align: center; }
      .rank-name { flex: 1; font-weight: 500; }
      .rank-pts { font-weight: 700; color: var(--color-primary); font-size: 1.1rem; }
      .rank-bar { position: absolute; bottom: 0; left: 0; height: 3px; background: var(--color-primary); border-radius: 0 4px 0 0; transition: width 0.5s ease; }

      .skeleton-list { display: flex; flex-direction: column; gap: 0.5rem; }
      .skel-row { height: 60px; background: var(--color-border); border-radius: 12px; animation: shimmer 1.5s infinite; background: linear-gradient(90deg, var(--color-border) 25%, var(--color-surface) 50%, var(--color-border) 75%); background-size: 200% 100%; }
      @keyframes shimmer { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }
      @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
      .empty-state { text-align: center; padding: 3rem; font-size: 1.2rem; color: var(--color-text-secondary); }
    </style>
  `
})
export class PoolComponent implements OnInit {
  poolId = ''; poolName = '';
  activeTab: Tab = 'matches';
  tabs: Tab[] = ['matches', 'bracket', 'ranking'];
  tabNames: Record<Tab, string> = { matches: '⚽ Partidos', bracket: '🏆 Bracket', ranking: '📊 Ranking' };

  // Teams
  teamMap: Record<string, TeamInfo> = {};
  teamsLoaded = false;

  // Matches
  matches: MatchItem[] = [];
  loadingMatches = true;
  preds: Record<string, { home: number; away: number }> = {};

  // Bracket: passed down to BracketViewComponent
  bracketTid = '019e4c4a-51f2-7b8c-9ea1-e492c1f08753';

  // Ranking
  ranking: any[] = [];
  loadingRanking = true;
  maxPts = 1;

  constructor(private route: ActivatedRoute, private api: ApiService, private toast: ToastService) {}

  ngOnInit() {
    this.poolId = this.route.snapshot.paramMap.get('id') || '';
    this.loadTeams();
    this.loadRanking();
  }

  loadTeams() {
    this.api.getTeams('019e4c4a-51f2-7b8c-9ea1-e492c1f08753').subscribe({
      next: (res) => {
        for (const t of res.teams) { this.teamMap[t.id] = t; }
        this.teamsLoaded = true;
        this.loadMatches();
      },
      error: () => this.loadMatches()
    });
  }

  loadMatches() {
    this.api.getMatches('019e4c4a-51f2-7b8c-9ea1-e492c1f08753').subscribe({
      next: (res) => {
        this.matches = res.matches || [];
        for (const m of this.matches) { this.preds[m.id] = { home: 0, away: 0 }; }
        this.loadingMatches = false;
      },
      error: () => this.loadingMatches = false
    });
  }

  getGroupLetter(gid: string): string { return gid.slice(-1); }

  stageLabel(s: string): string {
    const m: Record<string,string> = { group:'Grupos', round_of_32:'Octavos', round_of_16:'R16', quarter_final:'Cuartos', semi_final:'Semis', third_place:'3er Puesto', final:'Final' };
    return m[s] || s;
  }

  setTab(t: Tab) { this.activeTab = t; }

  savePred(matchId: string) {
    const p = this.preds[matchId];
    this.api.submitPrediction(this.poolId, matchId, p.home, p.away).subscribe({
      next: () => this.toast.success('Predicción guardada'),
      error: (err) => this.toast.error(err.error?.error || 'Error al guardar')
    });
  }

  loadRanking() {
    this.api.getRanking(this.poolId).subscribe({
      next: (res) => {
        this.ranking = res.ranking || [];
        if (this.ranking.length > 0) this.maxPts = this.ranking[0].total_points;
        this.loadingRanking = false;
      },
      error: () => this.loadingRanking = false
    });
  }
}
