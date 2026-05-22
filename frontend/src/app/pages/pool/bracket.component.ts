import { Component, Input, OnInit } from '@angular/core';
import { ApiService, BracketSlot, DerivedBracket, TeamInfo } from '../../services/api.service';
import { ToastService } from '../../services/toast.service';

const DEFAULT_TID = '019e4c4a-51f2-7b8c-9ea1-e492c1f08753';

@Component({
  selector: 'app-bracket-view',
  standalone: true,
  template: `
    @if (loading) {
      <div class="bracket-skeleton">
        <p class="hint">Cargando bracket…</p>
      </div>
    } @else {
      <div class="bracket-header">
        <p class="hint">
          Haz clic en el equipo que avanza en cada llave. Los ganadores se propagan automáticamente.
          El partido por el 3er puesto se llena con los perdedores de semifinales.
        </p>
        <div class="bracket-actions">
          <span class="status" [class.complete]="isComplete()">
            @if (isComplete()) { ✅ Bracket completo } @else { ⏳ Faltan {{ missingCount() }} ganador(es) }
          </span>
          <button class="btn-save-bracket" (click)="save()" [disabled]="!isComplete() || saving">
            @if (saving) { Guardando… } @else { 💾 Guardar bracket }
          </button>
        </div>
      </div>

      <div class="bracket-scroll">
        <div class="bracket-grid">
          <div class="round" [style.--rows]="16">
            <h3>Octavos (R32)</h3>
            @for (s of r32; track $index) {
              <div class="matchup" [class.empty]="!s.home_team_id">
                <button class="team-pick" [class.win]="s.winner_id === s.home_team_id" [class.lose]="s.winner_id && s.winner_id !== s.home_team_id"
                        [disabled]="!s.home_team_id" (click)="pick(0, $index, s.home_team_id)">
                  <span class="code">{{ labelFor(s.home_team_id) || s.home_label || '—' }}</span>
                </button>
                <button class="team-pick" [class.win]="s.winner_id === s.away_team_id" [class.lose]="s.winner_id && s.winner_id !== s.away_team_id"
                        [disabled]="!s.away_team_id" (click)="pick(0, $index, s.away_team_id)">
                  <span class="code">{{ labelFor(s.away_team_id) || s.away_label || '—' }}</span>
                </button>
              </div>
            }
          </div>

          <div class="round" [style.--rows]="8">
            <h3>Ronda de 16</h3>
            @for (s of r16; track $index) {
              <div class="matchup" [class.empty]="!s.home_team_id && !s.away_team_id">
                <button class="team-pick" [class.win]="s.winner_id === s.home_team_id" [class.lose]="s.winner_id && s.winner_id !== s.home_team_id"
                        [disabled]="!s.home_team_id" (click)="pick(1, $index, s.home_team_id)">
                  <span class="code">{{ labelFor(s.home_team_id) || '—' }}</span>
                </button>
                <button class="team-pick" [class.win]="s.winner_id === s.away_team_id" [class.lose]="s.winner_id && s.winner_id !== s.away_team_id"
                        [disabled]="!s.away_team_id" (click)="pick(1, $index, s.away_team_id)">
                  <span class="code">{{ labelFor(s.away_team_id) || '—' }}</span>
                </button>
              </div>
            }
          </div>

          <div class="round" [style.--rows]="4">
            <h3>Cuartos</h3>
            @for (s of qf; track $index) {
              <div class="matchup" [class.empty]="!s.home_team_id && !s.away_team_id">
                <button class="team-pick" [class.win]="s.winner_id === s.home_team_id" [class.lose]="s.winner_id && s.winner_id !== s.home_team_id"
                        [disabled]="!s.home_team_id" (click)="pick(2, $index, s.home_team_id)">
                  <span class="code">{{ labelFor(s.home_team_id) || '—' }}</span>
                </button>
                <button class="team-pick" [class.win]="s.winner_id === s.away_team_id" [class.lose]="s.winner_id && s.winner_id !== s.away_team_id"
                        [disabled]="!s.away_team_id" (click)="pick(2, $index, s.away_team_id)">
                  <span class="code">{{ labelFor(s.away_team_id) || '—' }}</span>
                </button>
              </div>
            }
          </div>

          <div class="round" [style.--rows]="2">
            <h3>Semifinal</h3>
            @for (s of sf; track $index) {
              <div class="matchup" [class.empty]="!s.home_team_id && !s.away_team_id">
                <button class="team-pick" [class.win]="s.winner_id === s.home_team_id" [class.lose]="s.winner_id && s.winner_id !== s.home_team_id"
                        [disabled]="!s.home_team_id" (click)="pick(3, $index, s.home_team_id)">
                  <span class="code">{{ labelFor(s.home_team_id) || '—' }}</span>
                </button>
                <button class="team-pick" [class.win]="s.winner_id === s.away_team_id" [class.lose]="s.winner_id && s.winner_id !== s.away_team_id"
                        [disabled]="!s.away_team_id" (click)="pick(3, $index, s.away_team_id)">
                  <span class="code">{{ labelFor(s.away_team_id) || '—' }}</span>
                </button>
              </div>
            }
          </div>

          <div class="round final-col" [style.--rows]="2">
            <h3>Final + 3er puesto</h3>
            <div class="matchup final-slot">
              <div class="slot-label">🥉 Tercer puesto</div>
              <button class="team-pick" [class.win]="third.winner_id === third.home_team_id" [class.lose]="third.winner_id && third.winner_id !== third.home_team_id"
                      [disabled]="!third.home_team_id" (click)="pickThird(third.home_team_id)">
                <span class="code">{{ labelFor(third.home_team_id) || '—' }}</span>
              </button>
              <button class="team-pick" [class.win]="third.winner_id === third.away_team_id" [class.lose]="third.winner_id && third.winner_id !== third.away_team_id"
                      [disabled]="!third.away_team_id" (click)="pickThird(third.away_team_id)">
                <span class="code">{{ labelFor(third.away_team_id) || '—' }}</span>
              </button>
            </div>
            <div class="matchup final-slot">
              <div class="slot-label">🏆 Final</div>
              <button class="team-pick" [class.win]="final.winner_id === final.home_team_id" [class.lose]="final.winner_id && final.winner_id !== final.home_team_id"
                      [disabled]="!final.home_team_id" (click)="pickFinal(final.home_team_id)">
                <span class="code">{{ labelFor(final.home_team_id) || '—' }}</span>
              </button>
              <button class="team-pick" [class.win]="final.winner_id === final.away_team_id" [class.lose]="final.winner_id && final.winner_id !== final.away_team_id"
                      [disabled]="!final.away_team_id" (click)="pickFinal(final.away_team_id)">
                <span class="code">{{ labelFor(final.away_team_id) || '—' }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      @if (final.winner_id) {
        <div class="champion-banner">🏆 Campeón previsto: <strong>{{ labelFor(final.winner_id) }}</strong></div>
      }
    }

    <style>
      .hint { font-size: 0.85rem; color: var(--color-text-secondary); margin-bottom: 0.6rem; line-height: 1.4; }
      .bracket-header { display: flex; flex-direction: column; gap: 0.6rem; margin-bottom: 1rem; }
      .bracket-actions { display: flex; align-items: center; gap: 1rem; flex-wrap: wrap; }
      .status { font-size: 0.85rem; color: var(--color-text-secondary); }
      .status.complete { color: var(--color-primary); font-weight: 700; }
      .btn-save-bracket {
        background: var(--color-primary); color: #fff; border: none;
        padding: 0.55rem 1.2rem; border-radius: 10px; font-weight: 600; cursor: pointer;
        transition: transform 0.15s, opacity 0.15s;
      }
      .btn-save-bracket:hover:not(:disabled) { transform: scale(1.02); }
      .btn-save-bracket:disabled { opacity: 0.45; cursor: not-allowed; }

      .bracket-scroll {
        overflow-x: auto;
        padding: 0.5rem 0 1rem;
        border-radius: 12px;
        background: var(--color-surface);
        box-shadow: inset 0 0 0 1px var(--color-border);
      }
      .bracket-grid { display: flex; gap: 1rem; padding: 1rem; min-width: max-content; }
      .round { display: flex; flex-direction: column; justify-content: space-around; gap: 0.5rem; min-width: 170px; }
      .round h3 {
        font-size: 0.78rem; text-transform: uppercase; color: var(--color-text-secondary);
        letter-spacing: 0.04em; margin-bottom: 0.4rem; text-align: center;
      }
      .final-col { gap: 1.5rem; }
      .matchup {
        display: flex; flex-direction: column; gap: 2px;
        background: var(--color-bg); border-radius: 8px; padding: 4px; box-shadow: 0 1px 3px rgba(0,0,0,0.05);
      }
      .matchup.empty { opacity: 0.35; }
      .slot-label { font-size: 0.72rem; color: var(--color-text-secondary); text-align: center; padding: 2px 0; }
      .team-pick {
        display: flex; align-items: center; justify-content: center;
        padding: 0.45rem 0.6rem; min-height: 34px; width: 100%;
        background: var(--color-surface); border: 1px solid var(--color-border);
        border-radius: 6px; cursor: pointer; font-family: inherit; font-size: 0.82rem; color: var(--color-text);
        transition: background 0.15s, transform 0.1s, border-color 0.15s;
        line-height: 1.15;
      }
      .team-pick .code { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%; }
      .team-pick:hover:not(:disabled) { background: var(--color-bg); border-color: var(--color-primary); }
      .team-pick:disabled { cursor: not-allowed; opacity: 0.5; }
      .team-pick.win { background: var(--color-primary); color: #fff; font-weight: 700; border-color: var(--color-primary); }
      .team-pick.lose { opacity: 0.45; text-decoration: line-through; }
      .code { font-weight: 600; letter-spacing: 0.02em; }

      .champion-banner {
        margin-top: 1rem; padding: 0.9rem 1.2rem;
        background: linear-gradient(135deg, var(--color-accent), var(--color-primary));
        color: #fff; border-radius: 12px; text-align: center; font-size: 1.05rem;
        box-shadow: 0 2px 10px rgba(0,0,0,0.1);
      }
    </style>
  `,
})
export class BracketViewComponent implements OnInit {
  @Input({ required: true }) poolId!: string;
  @Input() tournamentId: string = DEFAULT_TID;

  loading = true;
  saving = false;

  r32: BracketSlot[] = [];
  r16: BracketSlot[] = [];
  qf: BracketSlot[] = [];
  sf: BracketSlot[] = [];
  third: BracketSlot = emptySlot();
  final: BracketSlot = emptySlot();

  teamMap: Record<string, TeamInfo> = {};
  // The 32 teams that advance to R32 — derived once from groups and sent verbatim
  // to the backend (BracketPrediction requires them even though they're not user picks).
  private r32Teams: string[] = [];

  constructor(private api: ApiService, private toast: ToastService) {}

  ngOnInit() {
    this.api.getTeams(this.tournamentId).subscribe({
      next: (res) => {
        for (const t of res.teams) this.teamMap[t.id] = t;
        this.loadBracket();
      },
      error: () => this.loadBracket(),
    });
  }

  loadBracket() {
    this.api.getDerivedBracket(this.poolId, this.tournamentId).subscribe({
      next: (res) => {
        const b = res.bracket;
        this.r32 = (b.round_of_32 || []).map(cloneSlot);
        this.r16 = blankRound(8);
        this.qf = blankRound(4);
        this.sf = blankRound(2);
        this.third = emptySlot();
        this.final = emptySlot();
        // Seed any existing winner picks (in case the user already saved a bracket)
        seedPicks(this.r32, b.round_of_32);
        // Compute r32-derived teams to send on submit
        this.r32Teams = [];
        for (const s of this.r32) {
          this.r32Teams.push(s.home_team_id, s.away_team_id);
        }
        // Hydrate downstream rounds from the derived response so winner state is preserved
        this.r16 = (b.round_of_16 || []).map(cloneSlot);
        this.qf = (b.quarter_final || []).map(cloneSlot);
        this.sf = (b.semi_final || []).map(cloneSlot);
        this.third = cloneSlot(b.third_place || emptySlot());
        this.final = cloneSlot(b.final || emptySlot());
        // Ensure shapes
        if (this.r16.length < 8) this.r16 = padTo(this.r16, 8);
        if (this.qf.length < 4) this.qf = padTo(this.qf, 4);
        if (this.sf.length < 2) this.sf = padTo(this.sf, 2);
        // Recompute downstream slots from any existing winners to normalize state
        this.recomputeFrom(0);
        this.loading = false;
      },
      error: (err) => {
        this.loading = false;
        this.toast.error(err.error?.error || 'No se pudo cargar el bracket');
      },
    });
  }

  labelFor(id: string): string {
    if (!id) return '';
    const t = this.teamMap[id];
    return t?.name || t?.code || id.slice(0, 4).toUpperCase();
  }

  codeFor(id: string): string {
    if (!id) return '';
    return this.teamMap[id]?.code || id.slice(0, 4).toUpperCase();
  }

  pick(round: 0 | 1 | 2 | 3, slotIdx: number, teamId: string) {
    if (!teamId) return;
    const slots = this.roundAt(round);
    slots[slotIdx].winner_id = teamId;
    this.recomputeFrom(round + 1);
  }

  pickFinal(teamId: string) {
    if (!teamId) return;
    this.final.winner_id = teamId;
  }

  pickThird(teamId: string) {
    if (!teamId) return;
    this.third.winner_id = teamId;
  }

  // recomputeFrom rebuilds rounds[startRound..] from upstream winners.
  // Picks on a slot are cleared if the winner is no longer one of the new home/away.
  private recomputeFrom(startRound: number) {
    if (startRound <= 1) {
      this.r16 = rebuildNext(this.r32, this.r16, 8, 89);
    }
    if (startRound <= 2) {
      this.qf = rebuildNext(this.r16, this.qf, 4, 97);
    }
    if (startRound <= 3) {
      this.sf = rebuildNext(this.qf, this.sf, 2, 101);
    }
    if (startRound <= 4) {
      this.final = rebuildFinal(this.sf, this.final);
      this.third = rebuildThird(this.sf, this.third);
    }
  }

  private roundAt(idx: 0 | 1 | 2 | 3): BracketSlot[] {
    if (idx === 0) return this.r32;
    if (idx === 1) return this.r16;
    if (idx === 2) return this.qf;
    return this.sf;
  }

  isComplete(): boolean {
    if (this.r32.some((s) => !!s.home_team_id && !s.winner_id)) return false;
    if (this.r16.some((s) => !!s.home_team_id && !s.winner_id)) return false;
    if (this.qf.some((s) => !!s.home_team_id && !s.winner_id)) return false;
    if (this.sf.some((s) => !!s.home_team_id && !s.winner_id)) return false;
    if (!this.final.winner_id) return false;
    if (!this.third.winner_id) return false;
    return true;
  }

  missingCount(): number {
    let n = 0;
    for (const arr of [this.r32, this.r16, this.qf, this.sf]) {
      n += arr.filter((s) => !!s.home_team_id && !s.winner_id).length;
    }
    if (!this.final.winner_id) n++;
    if (!this.third.winner_id) n++;
    return n;
  }

  save() {
    if (!this.isComplete()) {
      this.toast.error('Completa todos los ganadores antes de guardar');
      return;
    }
    this.saving = true;
    const payload = {
      tournament_id: this.tournamentId,
      teams_to_round_of_32: this.r32Teams,
      teams_to_round_of_16: this.r32.map((s) => s.winner_id!),
      teams_to_quarter_final: this.r16.map((s) => s.winner_id!),
      teams_to_semi_final: this.qf.map((s) => s.winner_id!),
      teams_to_final: this.sf.map((s) => s.winner_id!),
      champion: this.final.winner_id!,
      third_place_winner: this.third.winner_id!,
    };
    this.api.submitBracket(this.poolId, payload).subscribe({
      next: () => {
        this.saving = false;
        this.toast.success('✅ Bracket guardado');
      },
      error: (err) => {
        this.saving = false;
        this.toast.error(err.error?.error || 'Error al guardar');
      },
    });
  }
}

function emptySlot(): BracketSlot {
  return { home_team_id: '', home_label: '', away_team_id: '', away_label: '' };
}

function cloneSlot(s: BracketSlot): BracketSlot {
  return { ...s };
}

function blankRound(n: number): BracketSlot[] {
  return Array.from({ length: n }, () => emptySlot());
}

function padTo(arr: BracketSlot[], n: number): BracketSlot[] {
  const out = arr.slice();
  while (out.length < n) out.push(emptySlot());
  return out;
}

function seedPicks(target: BracketSlot[], source: BracketSlot[] | undefined) {
  if (!source) return;
  for (let i = 0; i < target.length && i < source.length; i++) {
    if (source[i].winner_id) target[i].winner_id = source[i].winner_id;
  }
}

// rebuildNext takes the upstream round (from) and builds the next round (to)
// with home = winner of from[i*2], away = winner of from[i*2+1]. Preserves an
// existing winner_id only if it still matches one of the new sides.
function rebuildNext(from: BracketSlot[], previous: BracketSlot[], size: number, startMatchID: number): BracketSlot[] {
  const next: BracketSlot[] = [];
  for (let i = 0; i < size; i++) {
    const m1 = startMatchID + i * 2;
    const m2 = startMatchID + i * 2 + 1;
    const homeID = from[i * 2]?.winner_id || '';
    const awayID = from[i * 2 + 1]?.winner_id || '';
    const prevWinner = previous[i]?.winner_id || '';
    const winner = (prevWinner === homeID || prevWinner === awayID) ? prevWinner : '';
    next.push({
      home_team_id: homeID,
      home_label: `Ganador ${m1}`,
      away_team_id: awayID,
      away_label: `Ganador ${m2}`,
      winner_id: winner,
    });
  }
  return next;
}

function rebuildFinal(sf: BracketSlot[], previous: BracketSlot): BracketSlot {
  const home = sf[0]?.winner_id || '';
  const away = sf[1]?.winner_id || '';
  const prevWinner = previous?.winner_id || '';
  const winner = (prevWinner === home || prevWinner === away) ? prevWinner : '';
  return {
    home_team_id: home, home_label: 'Ganador 101',
    away_team_id: away, away_label: 'Ganador 102',
    winner_id: winner,
  };
}

function rebuildThird(sf: BracketSlot[], previous: BracketSlot): BracketSlot {
  const sf0 = sf[0];
  const sf1 = sf[1];
  const loser0 = loserOf(sf0);
  const loser1 = loserOf(sf1);
  const prevWinner = previous?.winner_id || '';
  const winner = (prevWinner === loser0 || prevWinner === loser1) ? prevWinner : '';
  return {
    home_team_id: loser0, home_label: 'Perdedor 101',
    away_team_id: loser1, away_label: 'Perdedor 102',
    winner_id: winner,
  };
}

function loserOf(s: BracketSlot | undefined): string {
  if (!s || !s.winner_id) return '';
  return s.winner_id === s.home_team_id ? s.away_team_id : s.home_team_id;
}
