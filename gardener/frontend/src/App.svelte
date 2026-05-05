<script>
  import { onMount, tick } from 'svelte'
  import PlantForm from './components/PlantForm.svelte'
  import PlantCard from './components/PlantCard.svelte'
  import ExportDialog from './components/ExportDialog.svelte'

  let plants = []
  let search = ''
  let viewMode = 'list' // 'list' | 'card' (полный просмотр одной записи)
  let displayMode = 'table' // 'table' | 'gallery' (отображение списка)
  let showForm = false
  let editingPlant = null
  let selectedFile = null
  let currentPlant = null
  let highlightedId = null
  let showExport = false

  async function load() {
    const res = await fetch('/api/plants')
    if (res.ok) plants = await res.json()
  }

  function openEdit(plant = null) {
    editingPlant = plant || {
      serial_number: '',
      plant_type: '',
      seed_source: '',
      description: '',
      fruit_description: '',
      technique: '',
      planting_years: '',
      comments: '',
      seed_remainder: 0,
      photo_path: '',
      tags: '',
      photos: '[]'
    }
    selectedFile = null
    showForm = true
    viewMode = 'list'
  }

  function openFullCard(plant) {
    currentPlant = plant
    viewMode = 'card'
    showForm = false
  }

  function closeFullCard() {
    viewMode = 'list'
    currentPlant = null
  }

  async function handleSave(data) {
    try {
      let res
      if (selectedFile) {
        const fd = new FormData()
        Object.entries(data).forEach(([key, value]) => {
          if (key !== 'photo' && value !== undefined && value !== null) {
            fd.append(key, String(value))
          }
        })
        fd.append('photo', selectedFile)
        res = await fetch('/api/plants', { method: 'POST', body: fd })
      } else {
        res = await fetch('/api/plants', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(data)
        })
      }

      if (!res.ok) {
        const err = await res.text()
        if (res.status === 409) {
          alert('⚠️ Запись с таким № наклейки уже существует')
        } else {
          alert('Ошибка: ' + (err || res.statusText))
        }
        return
      }

      showForm = false
      editingPlant = null
      selectedFile = null
      load()
    } catch (e) {
      alert('Ошибка сети: ' + e.message)
    }
  }

  async function deletePlant(plant) {
    if (!confirm(`Удалить "${plant.plant_type}" (#${plant.serial_number})?`)) return
    try {
      const res = await fetch(`/api/plants?serial=${encodeURIComponent(plant.serial_number)}`, {
        method: 'DELETE'
      })
      if (res.ok) {
        viewMode === 'card' ? closeFullCard() : null
        load()
      } else {
        alert('Ошибка удаления: ' + (await res.text()))
      }
    } catch (e) {
      alert('Ошибка сети: ' + e.message)
    }
  }

  async function handleSearch() {
    if (!search.trim()) {
      highlightedId = null
      return
    }
    const q = search.toLowerCase()
    const match = plants.find(
      (p) =>
        p.plant_type?.toLowerCase().includes(q) ||
        p.serial_number?.toLowerCase().includes(q) ||
        p.tags?.toLowerCase().includes(q)
    )
    if (match) {
      highlightedId = match.id
      await tick()
      const el =
        document.getElementById(`row-${match.id}`) || document.getElementById(`card-${match.id}`)
      el?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    } else {
      highlightedId = null
    }
  }

  function getThumb(p) {
    try {
      const arr = p.photos ? JSON.parse(p.photos) : []
      if (arr.length > 0) return arr[0]
    } catch {
      /* intentionally empty - ignore JSON parse errors */
    }
    return p.photo_path || null
  }

  onMount(load)
</script>

<main>
  {#if viewMode === 'card' && currentPlant}
    <PlantCard
      plant={currentPlant}
      onEdit={(p) => openEdit(p)}
      onDelete={deletePlant}
      onBack={closeFullCard}
    />
  {:else}
    <header class="app-header">
      <h1>🌱 Огородник</h1>
      <p class="subtitle">Учёт семян и агротехники</p>
    </header>

    <section class="toolbar">
      <div class="search-group">
        <input
          type="text"
          bind:value={search}
          placeholder="Поиск по виду, тегам, наклейке..."
          on:keydown={(e) => e.key === 'Enter' && handleSearch()}
        />
        <button on:click={handleSearch}>🔍 Найти</button>
      </div>
      <div class="actions-group">
        <button on:click={() => openEdit()}>➕ Добавить</button>
        <button
          class="toggle"
          on:click={() => (displayMode = displayMode === 'table' ? 'gallery' : 'table')}
        >
          {displayMode === 'table' ? '🖼️ Галерея' : '📋 Таблица'}
        </button>
        <button on:click={() => (showExport = true)}>📤 Экспорт</button>
      </div>
    </section>

    {#if displayMode === 'table'}
      <table class="data-table">
        <thead><tr><th>№</th><th>Вид</th><th>Теги</th><th>Остаток</th><th>Действия</th></tr></thead>
        <tbody>
          {#each plants as plant (plant.id)}
            <tr id="row-{plant.id}" class:highlighted={highlightedId === plant.id}>
              <td>{plant.serial_number}</td>
              <td class="cell-main">
                {#if getThumb(plant)}
                  <img
                    src={`/api/photos/${plant.serial_number}/${getThumb(plant)}`}
                    alt=""
                    class="thumb"
                    on:error={(e) => (e.target.style.display = 'none')}
                  />
                {/if}
                {plant.plant_type || '—'}
              </td>
              <td>{plant.tags || '—'}</td>
              <td>{plant.seed_remainder} шт</td>
              <td>
                <button class="icon-btn" on:click={() => openFullCard(plant)} title="Просмотр"
                  >👁️</button
                >
                <button class="icon-btn" on:click={() => openEdit(plant)} title="Редактировать"
                  >✏️</button
                >
                <button class="icon-btn danger" on:click={() => deletePlant(plant)} title="Удалить"
                  >🗑️</button
                >
              </td>
            </tr>
          {:else}
            <tr><td colspan="5">Нет записей</td></tr>
          {/each}
        </tbody>
      </table>
    {:else}
      <div class="gallery-grid">
        {#each plants as plant (plant.id)}
          <div
            id="card-{plant.id}"
            class="mini-card"
            class:highlighted={highlightedId === plant.id}
            on:click={() => openFullCard(plant)}
            on:keydown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                openFullCard(plant)
              }
            }}
            role="button"
            tabindex="0"
          >
            {#if getThumb(plant)}
              <img src={`/api/photos/${plant.serial_number}/${getThumb(plant)}`} alt="" />
            {:else}
              <div class="placeholder">📷</div>
            {/if}
            <div class="info">
              <h3>{plant.plant_type || 'Без названия'}</h3>
              <span>#{plant.serial_number}</span>
            </div>
          </div>
        {/each}
      </div>
    {/if}

    {#if showForm}
      <PlantForm
        plant={editingPlant}
        isNew={!editingPlant?.id}
        onSave={handleSave}
        onCancel={() => {
          showForm = false
          selectedFile = null
        }}
        onFileSelected={(file) => (selectedFile = file)}
      />
    {/if}
    <ExportDialog {plants} isOpen={showExport} onClose={() => (showExport = false)} />
  {/if}
</main>

<style>
  :root {
    --bg: #f8fafc;
    --card: #ffffff;
    --primary: #4f46e5;
    --text: #1e293b;
    --muted: #64748b;
    --border: #e2e8f0;
    --radius: 12px;
    --shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
  }
  main {
    max-width: 1200px;
    margin: 2rem auto;
    padding: 0 1.5rem;
    font-family:
      system-ui,
      -apple-system,
      sans-serif;
    color: var(--text);
    background: var(--bg);
    min-height: 100vh;
  }

  .app-header {
    text-align: center;
    margin-bottom: 1.5rem;
  }
  .app-header h1 {
    margin: 0 0 0.2rem;
    font-size: 2rem;
    font-weight: 700;
    color: var(--primary);
  }
  .subtitle {
    margin: 0;
    color: var(--muted);
    font-size: 0.95rem;
  }

  .toolbar {
    background: var(--card);
    padding: 1rem;
    border-radius: var(--radius);
    box-shadow: var(--shadow);
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    align-items: center;
    margin-bottom: 1.5rem;
  }
  .search-group {
    flex: 1;
    display: flex;
    gap: 0.5rem;
    min-width: 250px;
  }
  .actions-group {
    display: flex;
    gap: 0.5rem;
  }
  input,
  button {
    padding: 0.6rem 1rem;
    border: 1px solid var(--border);
    border-radius: 8px;
    font-size: 0.95rem;
    background: white;
  }
  button {
    cursor: pointer;
    background: var(--primary);
    color: white;
    border: none;
    font-weight: 500;
  }
  button.toggle {
    background: white;
    color: var(--text);
    border: 1px solid var(--border);
  }
  button:hover {
    opacity: 0.9;
    transform: translateY(-1px);
  }

  .data-table {
    width: 100%;
    border-collapse: separate;
    border-spacing: 0;
    background: var(--card);
    border-radius: var(--radius);
    overflow: hidden;
    box-shadow: var(--shadow);
  }
  .data-table th,
  .data-table td {
    padding: 0.9rem 1rem;
    text-align: left;
    border-bottom: 1px solid var(--border);
  }
  .data-table th {
    background: #f1f5f9;
    font-weight: 600;
    color: var(--muted);
    text-transform: uppercase;
    font-size: 0.75rem;
    letter-spacing: 0.05em;
  }
  .cell-main {
    display: flex;
    align-items: center;
    gap: 0.6rem;
  }
  .thumb {
    width: 40px;
    height: 40px;
    object-fit: cover;
    border-radius: 6px;
    background: #eee;
  }
  .icon-btn {
    background: transparent;
    border: 1px solid var(--border);
    padding: 0.4rem;
    border-radius: 6px;
    font-size: 1rem;
  }
  .icon-btn.danger {
    color: #ef4444;
    border-color: #fecaca;
  }
  .icon-btn:hover {
    background: #f8fafc;
  }

  .gallery-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 1rem;
  }
  .mini-card {
    background: var(--card);
    border-radius: var(--radius);
    overflow: hidden;
    box-shadow: var(--shadow);
    cursor: pointer;
    transition:
      transform 0.2s,
      box-shadow 0.2s;
  }
  .mini-card:hover {
    transform: translateY(-3px);
    box-shadow: 0 8px 16px rgba(0, 0, 0, 0.1);
  }
  .mini-card img,
  .mini-card .placeholder {
    width: 100%;
    height: 140px;
    object-fit: cover;
    background: #f1f5f9;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 2rem;
    color: var(--muted);
  }
  .mini-card .info {
    padding: 0.8rem;
  }
  .mini-card h3 {
    margin: 0 0 0.3rem;
    font-size: 1rem;
  }
  .mini-card span {
    color: var(--muted);
    font-size: 0.85rem;
  }

  .highlighted {
    background-color: #fef9c3 !important;
    transition: background 0.3s;
  }
</style>
