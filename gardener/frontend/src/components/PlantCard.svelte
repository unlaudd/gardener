<script>
  export let plant = {}
  export let onEdit
  export let onDelete
  export let onBack

  let gallery = []
  let currentIdx = 0

  $: {
    try {
      gallery = plant.photos ? JSON.parse(plant.photos) : []
    } catch {
      gallery = []
    }
    if (gallery.length === 0 && plant.photo_path) gallery = [plant.photo_path]
    currentIdx = 0
  }

  function prev() {
    if (gallery.length) currentIdx = (currentIdx - 1 + gallery.length) % gallery.length
  }
  function next() {
    if (gallery.length) currentIdx = (currentIdx + 1) % gallery.length
  }
</script>

<div class="card-wrapper">
  <div class="card">
    <header class="card-header">
      <h2>{plant.plant_type || 'Без названия'}</h2>
      <span class="serial">#{plant.serial_number}</span>
    </header>

    {#if gallery.length > 0}
      <div class="photo-gallery">
        <button class="nav" on:click={prev}>←</button>
        <img src={`/api/photos/${plant.serial_number}/${gallery[currentIdx]}`} alt="Фото" />
        <button class="nav right" on:click={next}>→</button>
        <div class="counter">{currentIdx + 1} / {gallery.length}</div>
      </div>
    {:else}
      <div class="photo-gallery placeholder">📷 Нет фото</div>
    {/if}

    <section class="info-grid">
      <div class="field"><strong>Вид растения:</strong> {plant.plant_type || '—'}</div>
      <div class="field"><strong>Откуда семена:</strong> {plant.seed_source || '—'}</div>
      <div class="field"><strong>Описание:</strong> {plant.description || '—'}</div>
      <div class="field"><strong>Описание плодов:</strong> {plant.fruit_description || '—'}</div>
      <div class="field"><strong>Агротехника:</strong> {plant.technique || '—'}</div>
      <div class="field"><strong>Годы посадки:</strong> {plant.planting_years || '—'}</div>
      <div class="field"><strong>Остаток семян:</strong> {plant.seed_remainder} шт</div>
      <div class="field"><strong>Теги:</strong> {plant.tags || '—'}</div>
      <div class="field full"><strong>Комментарии:</strong> {plant.comments || '—'}</div>
    </section>

    <footer class="actions">
      <button class="primary" on:click={() => onEdit(plant)}>✏️ Редактировать</button>
      <button class="danger" on:click={() => onDelete(plant)}>🗑️ Удалить</button>
      <button on:click={onBack}>⬅️ Назад к списку</button>
    </footer>
  </div>
</div>

<style>
  .card-wrapper {
    max-width: 800px;
    margin: 2rem auto;
  }
  .card {
    background: white;
    border-radius: 16px;
    box-shadow: 0 8px 30px rgba(0, 0, 0, 0.08);
    overflow: hidden;
  }
  .card-header {
    background: linear-gradient(135deg, #4f46e5, #7c3aed);
    color: white;
    padding: 1.5rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .card-header h2 {
    margin: 0;
    font-size: 1.5rem;
    font-weight: 600;
  }
  .serial {
    background: rgba(255, 255, 255, 0.25);
    padding: 0.4rem 1rem;
    border-radius: 20px;
    font-size: 0.9rem;
    font-weight: 500;
  }

  .photo-gallery {
    position: relative;
    background: #f8fafc;
    padding: 2rem 1rem;
    text-align: center;
    min-height: 280px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .photo-gallery img {
    max-width: 100%;
    max-height: 350px;
    object-fit: contain;
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  }
  .photo-gallery.placeholder {
    color: #64748b;
    font-size: 1.3rem;
  }
  .nav {
    position: absolute;
    top: 50%;
    transform: translateY(-50%);
    background: rgba(255, 255, 255, 0.9);
    color: #1e293b;
    border: none;
    width: 44px;
    height: 44px;
    border-radius: 50%;
    font-size: 1.2rem;
    cursor: pointer;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
    z-index: 2;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .nav:hover {
    background: white;
    transform: translateY(-50%) scale(1.1);
  }
  .nav.right {
    left: auto;
    right: 1rem;
  }
  .nav:first-child {
    left: 1rem;
  }
  .counter {
    position: absolute;
    bottom: 12px;
    right: 12px;
    background: rgba(0, 0, 0, 0.7);
    color: white;
    padding: 0.3rem 0.8rem;
    border-radius: 20px;
    font-size: 0.8rem;
    pointer-events: none;
  }

  .info-grid {
    padding: 1.5rem 2rem;
    display: grid;
    gap: 1.2rem;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  }
  .field {
    line-height: 1.6;
    color: #334155;
  }
  .field.full {
    grid-column: 1/-1;
  }
  .field strong {
    color: #64748b;
    font-weight: 600;
    display: block;
    font-size: 0.85rem;
    margin-bottom: 0.2rem;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .actions {
    display: flex;
    gap: 0.8rem;
    padding: 1.2rem 2rem;
    background: #f8fafc;
    border-top: 1px solid #e2e8f0;
    flex-wrap: wrap;
  }
  button {
    padding: 0.7rem 1.4rem;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.95rem;
    font-weight: 500;
  }
  button.primary {
    background: #4f46e5;
    color: white;
  }
  button.danger {
    background: #ef4444;
    color: white;
  }
  button:hover {
    opacity: 0.9;
    transform: translateY(-1px);
  }
</style>
