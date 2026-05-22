import { Component, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { ApiService, MatchItem, RankingEntry } from '../../services/api.service';
import { NavbarComponent } from '../../layout/navbar.component';

type Tab = 'matches' | 'bracket' | 'ranking';

@Component({
  selector: 'app-pool',
  standalone: true,
  imports: [FormsModule, RouterModule, NavbarComponent],
  template: `
    <app-navbar />
    <div class="container">
      <a routerLink="/dashboard" class="back">← Volver a mis grupos</a>
      <h1>{{ poolName || 'Grupo' }}</h1>

      <div class="tabs">
        @for (t of tabs; track t) {
          <button [class.active]="activeTab === t" (click)="setTab(t)">{{ tabNames[t] }}</button>
        }
      </div>

      @if (activeTab === 'matches') {
        @if (loadingMatches) { <p class="loading">Cargando partidos...</p> }
        @else {
          <div class="match-list">
            @for (m of matches; track m.id) {
              <div class="card match-card" [class.finished]="m.status === 'finished'">
                <div class="match-info">
                  <span class="stage">{{ m.stage }}</span>
                  @if (m.group_id) { <span class="group">Grupo {{ m.group_id?.slice(-1) }}</span> }
                  <span class="kickoff">{{ m.kickoff_at | date:'dd/MM HH:mm' }}</span>
                  <span class="venue">{{ m.venue }}</span>
                </div>
                <div class="prediction-row">
                  <input type="number" min="0" max="30" [(ngModel)]="preds[m.id].home"
                         [disabled]="m.status === 'finished'" class="goal-input" />
                  <span class="team-id">{{ m.home_team_id.slice(0,8) }}</span>
                  <span class="vs">vs</span>
                  <span class="team-id">{{ m.away_team_id.slice(0,8) }}</span>
                  <input type="number" min="0" max="30" [(ngModel)]="preds[m.id].away"
                         [disabled]="m.status === 'finished'" class="goal-input" />
                  <button class="btn-save" [disabled]="m.status === 'finished'"
                          (click)="savePrediction(m.id)">Guardar</button>
                </div>
                @if (saveMsgs[m.id]) { <p class="save-msg">{{ saveMsgs[m.id] }}</p> }
              </div>
            }
          </div>
        }
      }

      @if (activeTab === 'bracket') {
        <div class="card">
          <h2>Pronóstico de Bracket</h2>
          <p class="hint">Selecciona 32 equipos para octavos, 16 para ronda de 16, 8 para cuartos, 4 para semis, 2 finalistas, campeón y tercer puesto. Los equipos deben respetar la cascada.</p>
          <label>ID del Torneo</label>
          <input [(ngModel)]="bracketTournamentId" class="full-input" />
          <label>Octavos (32 IDs, separados por coma)</label>
          <textarea [(ngModel)]="r32Str" rows="2" class="full-input"></textarea>
          <label>Ronda de 16 (16 IDs)</label>
          <textarea [(ngModel)]="r16Str" rows="2" class="full-input"></textarea>
          <label>Cuartos (8 IDs)</label>
          <textarea [(ngModel)]="qfStr" rows="1" class="full-input"></textarea>
          <label>Semifinal (4 IDs)</label>
          <textarea [(ngModel)]="sfStr" rows="1" class="full-input"></textarea>
          <label>Final (2 IDs)</label>
          <textarea [(ngModel)]="fStr" rows="1" class="full-input"></textarea>
          <label>Campeón (1 ID)</label>
          <input [(ngModel)]="champion" class="full-input" />
          <label>Tercer puesto (1 ID)</label>
          <input [(ngModel)]="thirdPlace" class="full-input" />
          @if (bracketError) { <p class="error">{{ bracketError }}</p> }
          @if (bracketSuccess) { <p class="success">{{ bracketSuccess }}</p> }
          <button class="btn-primary" (click)="saveBracket()">Guardar Bracket</button>
        </div>
      }

      @if (activeTab === 'ranking') {
        @if (loadingRanking) { <p class="loading">Cargando ranking...</p> }
        @else if (ranking.length === 0) { <p class="empty">Sin puntajes todavía.</p> }
        @else {
          <table class="ranking-table">
            <thead>
              <tr><th>#</th><th>Usuario</th><th>Puntos</th></tr>
            </thead>
            <tbody>
              @for (r of ranking; track r.user_id; let i = $index) {
                <tr [class.top3]="i < 3">
                  <td>{{ i + 1 }}</td>
                  <td>{{ r.display_name }}</td>
                  <td class="pts">{{ r.total_points }}</td>
                </tr>
              }
            </tbody>
          </table>
        }
      }
    </div>

    <style>
      .container { max-width: 900px; margin: 0 auto; padding: 1.5rem 1rem; }
      .back { color: var(--color-primary); text-decoration: none; font-size: 0.9rem; }
      h1 { margin: 0.5rem 0 1rem; color: var(--color-text); }
      .tabs { display: flex; gap: 0; margin-bottom: 1.5rem; }
      .tabs button {
        padding: 0.5rem 1.2rem; border: 1px solid var(--color-border);
        background: var(--color-surface); color: var(--color-text); cursor: pointer; font-size: 0.9rem;
      }
      .tabs button:first-child { border-radius: 8px 0 0 8px; }
      .tabs button:last-child { border-radius: 0 8px 8px 0; }
      .tabs button.active { background: var(--color-primary); color: #fff; border-color: var(--color-primary); }

      .card { background: var(--color-surface); border-radius: 12px; padding: 1.5rem; margin-bottom: 0.8rem; box-shadow: 0 2px 8px rgba(0,0,0,0.04); }
      .match-card.finished { opacity: 0.6; }
      .match-info { display: flex; gap: 1rem; font-size: 0.8rem; color: var(--color-text-secondary); margin-bottom: 0.5rem; }
      .prediction-row { display: flex; align-items: center; gap: 0.5rem; }
      .goal-input { width: 48px; padding: 0.4rem; text-align: center; border: 1px solid var(--color-border); border-radius: 6px; background: var(--color-bg); color: var(--color-text); }
      .team-id { font-size: 0.8rem; color: var(--color-text-secondary); }
      .vs { font-weight: 600; font-size: 0.85rem; }
      .btn-save { background: var(--color-primary); color: #fff; border: none; padding: 0.4rem 0.8rem; border-radius: 6px; cursor: pointer; font-size: 0.8rem; }
      .btn-save:disabled { opacity: 0.5; cursor: not-allowed; }
      .save-msg { font-size: 0.75rem; color: var(--color-success); margin-top: 0.2rem; }

      .full-input { width: 100%; padding: 0.5rem; border: 1px solid var(--color-border); border-radius: 8px; background: var(--color-bg); color: var(--color-text); margin-bottom: 0.8rem; }
      textarea.full-input { resize: vertical; font-family: monospace; font-size: 0.8rem; }
      .hint { font-size: 0.85rem; color: var(--color-text-secondary); margin-bottom: 1rem; }
      .btn-primary { background: var(--color-primary); color: #fff; border: none; padding: 0.6rem 1.5rem; border-radius: 8px; font-weight: 600; cursor: pointer; }

      .ranking-table { width: 100%; border-collapse: collapse; }
      .ranking-table th, .ranking-table td { padding: 0.7rem 1rem; text-align: left; border-bottom: 1px solid var(--color-border); }
      .ranking-table th { font-size: 0.85rem; color: var(--color-text-secondary); }
      .top3 { font-weight: 600; }
      .pts { font-weight: 700; color: var(--color-primary); }

      .loading, .empty { text-align: center; color: var(--color-text-secondary); padding: 3rem; }
      .error { color: var(--color-danger); font-size: 0.85rem; margin-top: 0.5rem; }
      .success { color: var(--color-success); font-size: 0.85rem; margin-top: 0.5rem; }
    </style>
  `
})
export class PoolComponent implements OnInit {
  poolId = '';
  poolName = '';
  activeTab: Tab = 'matches';
  tabs: Tab[] = ['matches', 'bracket', 'ranking'];
  tabNames: Record<Tab, string> = { matches: 'Partidos', bracket: 'Bracket', ranking: 'Ranking' };

  // Matches
  matches: MatchItem[] = [];
  loadingMatches = true;
  preds: Record<string, { home: number; away: number }> = {};
  saveMsgs: Record<string, string> = {};

  // Bracket
  bracketTournamentId = '';
  r32Str = ''; r16Str = ''; qfStr = ''; sfStr = ''; fStr = '';
  champion = ''; thirdPlace = '';
  bracketError = ''; bracketSuccess = '';

  // Ranking
  ranking: RankingEntry[] = [];
  loadingRanking = true;

  constructor(private route: ActivatedRoute, private api: ApiService) {}

  ngOnInit() {
    this.poolId = this.route.snapshot.paramMap.get('id') || '';
    this.loadMatches();
    this.loadRanking();
  }

  setTab(t: Tab) { this.activeTab = t; }

  loadMatches() {
    this.loadingMatches = true;
    this.api.getMatches('019e4c4a-51f2-7b8c-9ea1-e492c1f08753').subscribe({
      next: (res) => {
        this.matches = res.matches || [];
        for (const m of this.matches) { this.preds[m.id] = { home: 0, away: 0 }; }
        this.loadingMatches = false;
      },
      error: () => { this.loadingMatches = false; }
    });
  }

  savePrediction(matchId: string) {
    const p = this.preds[matchId];
    this.api.submitPrediction(this.poolId, matchId, p.home, p.away).subscribe({
      next: () => { this.saveMsgs[matchId] = '✅ Guardado'; setTimeout(() => delete this.saveMsgs[matchId], 2000); },
      error: (err) => { this.saveMsgs[matchId] = '❌ ' + (err.error?.error || 'Error'); }
    });
  }

  saveBracket() {
    this.bracketError = ''; this.bracketSuccess = '';
    const parse = (s: string) => s.split(',').map(t => t.trim()).filter(t => t);
    const r32 = parse(this.r32Str), r16 = parse(this.r16Str), qf = parse(this.qfStr);
    const sf = parse(this.sfStr), f = parse(this.fStr);

    this.api.submitBracket(this.poolId, {
      tournament_id: this.bracketTournamentId,
      teams_to_round_of_32: r32, teams_to_round_of_16: r16,
      teams_to_quarter_final: qf, teams_to_semi_final: sf,
      teams_to_final: f, champion: this.champion.trim(), third_place_winner: this.thirdPlace.trim()
    }).subscribe({
      next: () => { this.bracketSuccess = '✅ Bracket guardado correctamente.'; },
      error: (err) => { this.bracketError = err.error?.error || 'Error al guardar.'; }
    });
  }

  loadRanking() {
    this.loadingRanking = true;
    this.api.getRanking(this.poolId).subscribe({
      next: (res) => { this.ranking = res.ranking || []; this.loadingRanking = false; },
      error: () => { this.loadingRanking = false; }
    });
  }
}
