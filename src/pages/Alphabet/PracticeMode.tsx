// PracticeMode.tsx - Modern Interactive Practice
import React, { useState, useEffect } from 'react';
import styles from './PracticeMode.module.css';

// Practice data structure
const practiceData = {
  hiragana: [
    { character: 'あ', romaji: 'a', sound: 'a' },
    { character: 'い', romaji: 'i', sound: 'i' },
    { character: 'う', romaji: 'u', sound: 'u' },
    { character: 'え', romaji: 'e', sound: 'e' },
    { character: 'お', romaji: 'o', sound: 'o' },
    { character: 'か', romaji: 'ka', sound: 'ka' },
    { character: 'き', romaji: 'ki', sound: 'ki' },
    { character: 'く', romaji: 'ku', sound: 'ku' },
    { character: 'け', romaji: 'ke', sound: 'ke' },
    { character: 'こ', romaji: 'ko', sound: 'ko' },
    { character: 'さ', romaji: 'sa', sound: 'sa' },
    { character: 'し', romaji: 'shi', sound: 'shi' },
    { character: 'す', romaji: 'su', sound: 'su' },
    { character: 'せ', romaji: 'se', sound: 'se' },
    { character: 'そ', romaji: 'so', sound: 'so' },
    { character: 'た', romaji: 'ta', sound: 'ta' },
    { character: 'ち', romaji: 'chi', sound: 'chi' },
    { character: 'つ', romaji: 'tsu', sound: 'tsu' },
    { character: 'て', romaji: 'te', sound: 'te' },
    { character: 'と', romaji: 'to', sound: 'to' },
    { character: 'な', romaji: 'na', sound: 'na' },
    { character: 'に', romaji: 'ni', sound: 'ni' },
    { character: 'ぬ', romaji: 'nu', sound: 'nu' },
    { character: 'ね', romaji: 'ne', sound: 'ne' },
    { character: 'の', romaji: 'no', sound: 'no' },
    { character: 'は', romaji: 'ha', sound: 'ha' },
    { character: 'ひ', romaji: 'hi', sound: 'hi' },
    { character: 'ふ', romaji: 'fu', sound: 'fu' },
    { character: 'へ', romaji: 'he', sound: 'he' },
    { character: 'ほ', romaji: 'ho', sound: 'ho' },
    { character: 'ま', romaji: 'ma', sound: 'ma' },
    { character: 'み', romaji: 'mi', sound: 'mi' },
    { character: 'む', romaji: 'mu', sound: 'mu' },
    { character: 'め', romaji: 'me', sound: 'me' },
    { character: 'も', romaji: 'mo', sound: 'mo' },
    { character: 'や', romaji: 'ya', sound: 'ya' },
    { character: 'ゆ', romaji: 'yu', sound: 'yu' },
    { character: 'よ', romaji: 'yo', sound: 'yo' },
    { character: 'ら', romaji: 'ra', sound: 'ra' },
    { character: 'り', romaji: 'ri', sound: 'ri' },
    { character: 'る', romaji: 'ru', sound: 'ru' },
    { character: 'れ', romaji: 're', sound: 're' },
    { character: 'ろ', romaji: 'ro', sound: 'ro' },
    { character: 'わ', romaji: 'wa', sound: 'wa' },
    { character: 'を', romaji: 'wo', sound: 'wo' },
    { character: 'ん', romaji: 'n', sound: 'n' }
  ],
  katakana: [
    { character: 'ア', romaji: 'a', sound: 'a' },
    { character: 'イ', romaji: 'i', sound: 'i' },
    { character: 'ウ', romaji: 'u', sound: 'u' },
    { character: 'エ', romaji: 'e', sound: 'e' },
    { character: 'オ', romaji: 'o', sound: 'o' },
    { character: 'カ', romaji: 'ka', sound: 'ka' },
    { character: 'キ', romaji: 'ki', sound: 'ki' },
    { character: 'ク', romaji: 'ku', sound: 'ku' },
    { character: 'ケ', romaji: 'ke', sound: 'ke' },
    { character: 'コ', romaji: 'ko', sound: 'ko' },
    { character: 'サ', romaji: 'sa', sound: 'sa' },
    { character: 'シ', romaji: 'shi', sound: 'shi' },
    { character: 'ス', romaji: 'su', sound: 'su' },
    { character: 'セ', romaji: 'se', sound: 'se' },
    { character: 'ソ', romaji: 'so', sound: 'so' },
    { character: 'タ', romaji: 'ta', sound: 'ta' },
    { character: 'チ', romaji: 'chi', sound: 'chi' },
    { character: 'ツ', romaji: 'tsu', sound: 'tsu' },
    { character: 'テ', romaji: 'te', sound: 'te' },
    { character: 'ト', romaji: 'to', sound: 'to' },
    { character: 'ナ', romaji: 'na', sound: 'na' },
    { character: 'ニ', romaji: 'ni', sound: 'ni' },
    { character: 'ヌ', romaji: 'nu', sound: 'nu' },
    { character: 'ネ', romaji: 'ne', sound: 'ne' },
    { character: 'ノ', romaji: 'no', sound: 'no' },
    { character: 'ハ', romaji: 'ha', sound: 'ha' },
    { character: 'ヒ', romaji: 'hi', sound: 'hi' },
    { character: 'フ', romaji: 'fu', sound: 'fu' },
    { character: 'ヘ', romaji: 'he', sound: 'he' },
    { character: 'ホ', romaji: 'ho', sound: 'ho' },
    { character: 'マ', romaji: 'ma', sound: 'ma' },
    { character: 'ミ', romaji: 'mi', sound: 'mi' },
    { character: 'ム', romaji: 'mu', sound: 'mu' },
    { character: 'メ', romaji: 'me', sound: 'me' },
    { character: 'モ', romaji: 'mo', sound: 'mo' },
    { character: 'ヤ', romaji: 'ya', sound: 'ya' },
    { character: 'ユ', romaji: 'yu', sound: 'yu' },
    { character: 'ヨ', romaji: 'yo', sound: 'yo' },
    { character: 'ラ', romaji: 'ra', sound: 'ra' },
    { character: 'リ', romaji: 'ri', sound: 'ri' },
    { character: 'ル', romaji: 'ru', sound: 'ru' },
    { character: 'レ', romaji: 're', sound: 're' },
    { character: 'ロ', romaji: 'ro', sound: 'ro' },
    { character: 'ワ', romaji: 'wa', sound: 'wa' },
    { character: 'ヲ', romaji: 'wo', sound: 'wo' },
    { character: 'ン', romaji: 'n', sound: 'n' }
  ]
};

interface Props {
  type: 'hiragana' | 'katakana';
  onExit: () => void;
}

type QuizMode = 'sound-to-char' | 'char-to-sound' | 'romaji-to-char' | 'char-to-romaji';
type QuizState = 'setup' | 'active' | 'results';

interface Question {
  correct: any;
  options: any[];
  question: string;
  type: string;
}

const PracticeMode: React.FC<Props> = ({ type, onExit }) => {
  // State management
  const [quizState, setQuizState] = useState<QuizState>('setup');
  const [quizMode, setQuizMode] = useState<QuizMode>('sound-to-char');
  const [questionCount, setQuestionCount] = useState(10);
  const [selectedRows, setSelectedRows] = useState<string[]>(['あ行', 'か行']);
  
  // Quiz progress
  const [currentQuestion, setCurrentQuestion] = useState(0);
  const [score, setScore] = useState(0);
  const [questions, setQuestions] = useState<Question[]>([]);
  const [selectedAnswer, setSelectedAnswer] = useState<string>('');
  const [showAnswer, setShowAnswer] = useState(false);
  const [timeLeft, setTimeLeft] = useState(15);

  const availableRows = type === 'hiragana' 
    ? ['あ行', 'か行', 'さ行', 'た行', 'な行', 'は行', 'ま行', 'や行', 'ら行', 'わ行']
    : ['ア行', 'カ行', 'サ行', 'タ行', 'ナ行', 'ハ行', 'マ行', 'ヤ行', 'ラ行', 'ワ行'];

  // Timer effect
  useEffect(() => {
    let timer: NodeJS.Timeout;
    if (quizState === 'active' && !showAnswer && timeLeft > 0) {
      timer = setTimeout(() => setTimeLeft(timeLeft - 1), 1000);
    } else if (timeLeft === 0 && !showAnswer) {
      handleAnswer('');
    }
    return () => clearTimeout(timer);
  }, [quizState, timeLeft, showAnswer]);

  // Generate questions
  const generateQuestions = () => {
    const data = practiceData[type];
    const filteredData = data.filter(char => {
      const rowMap: { [key: string]: string[] } = {
        'あ行': ['あ', 'い', 'う', 'え', 'お'],
        'ア行': ['ア', 'イ', 'ウ', 'エ', 'オ'],
        'か行': ['か', 'き', 'く', 'け', 'こ'],
        'カ行': ['カ', 'キ', 'ク', 'ケ', 'コ'],
        'さ行': ['さ', 'し', 'す', 'せ', 'そ'],
        'サ行': ['サ', 'シ', 'ス', 'セ', 'ソ'],
        'た行': ['た', 'ち', 'つ', 'て', 'と'],
        'タ行': ['タ', 'チ', 'ツ', 'テ', 'ト'],
        'な行': ['な', 'に', 'ぬ', 'ね', 'の'],
        'ナ行': ['ナ', 'ニ', 'ヌ', 'ネ', 'ノ'],
        'は行': ['は', 'ひ', 'ふ', 'へ', 'ほ'],
        'ハ行': ['ハ', 'ヒ', 'フ', 'ヘ', 'ホ'],
        'ま行': ['ま', 'み', 'む', 'め', 'も'],
        'マ行': ['マ', 'ミ', 'ム', 'メ', 'モ'],
        'や行': ['や', 'ゆ', 'よ'],
        'ヤ行': ['ヤ', 'ユ', 'ヨ'],
        'ら行': ['ら', 'り', 'る', 'れ', 'ろ'],
        'ラ行': ['ラ', 'リ', 'ル', 'レ', 'ロ'],
        'わ行': ['わ', 'を', 'ん'],
        'ワ行': ['ワ', 'ヲ', 'ン']
      };
      
      return selectedRows.some(row => 
        rowMap[row]?.includes(char.character)
      );
    });

    const newQuestions: Question[] = [];
    for (let i = 0; i < questionCount; i++) {
      const correct = filteredData[Math.floor(Math.random() * filteredData.length)];
      let options = [correct];
      
      // Add 3 wrong options
      while (options.length < 4) {
        const random = filteredData[Math.floor(Math.random() * filteredData.length)];
        if (!options.find(opt => opt.character === random.character)) {
          options.push(random);
        }
      }
      
      // Shuffle options
      options = options.sort(() => Math.random() - 0.5);
      
      let question = '';
      switch (quizMode) {
        case 'sound-to-char':
          question = `Which character makes the "${correct.romaji}" sound?`;
          break;
        case 'char-to-sound':
          question = `What sound does "${correct.character}" make?`;
          break;
        case 'romaji-to-char':
          question = `Which character represents "${correct.romaji}"?`;
          break;
        case 'char-to-romaji':
          question = `What is the romaji for "${correct.character}"?`;
          break;
      }
      
      newQuestions.push({
        correct,
        options,
        question,
        type: quizMode
      });
    }
    
    setQuestions(newQuestions);
  };

  const startQuiz = () => {
    generateQuestions();
    setQuizState('active');
    setCurrentQuestion(0);
    setScore(0);
    setTimeLeft(15);
    setSelectedAnswer('');
    setShowAnswer(false);
  };

  const handleAnswer = (answer: string) => {
    setSelectedAnswer(answer);
    setShowAnswer(true);
    
    if (answer === questions[currentQuestion].correct.character || 
        answer === questions[currentQuestion].correct.romaji) {
      setScore(score + 1);
    }
    
    setTimeout(() => {
      if (currentQuestion + 1 < questions.length) {
        setCurrentQuestion(currentQuestion + 1);
        setSelectedAnswer('');
        setShowAnswer(false);
        setTimeLeft(15);
      } else {
        setQuizState('results');
      }
    }, 2000);
  };

  const resetQuiz = () => {
    setQuizState('setup');
    setCurrentQuestion(0);
    setScore(0);
    setSelectedAnswer('');
    setShowAnswer(false);
  };

  // Setup Screen
  if (quizState === 'setup') {
    return (
      <div className={styles.practiceMode}>
        <div className={styles.setupContainer}>
          <div className={styles.setupHeader}>
            <div className={styles.setupTitle}>
              <span className={styles.titleIcon}>{type === 'hiragana' ? '🔤' : '🔡'}</span>
              <h2>Practice {type === 'hiragana' ? 'Hiragana' : 'Katakana'}</h2>
            </div>
            <button className={styles.exitBtn} onClick={onExit}>
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M18 6L6 18M6 6l12 12" stroke="currentColor" strokeWidth="2"/>
              </svg>
            </button>
          </div>

          <div className={styles.setupGrid}>
            {/* Quiz Mode Selection */}
            <div className={styles.setupSection}>
              <h3 className={styles.sectionTitle}>
                <span className={styles.sectionIcon}>🎯</span>
                Quiz Mode
              </h3>
              <div className={styles.radioGroup}>
                <label className={styles.radioOption}>
                  <input
                    type="radio"
                    name="quizMode"
                    value="sound-to-char"
                    checked={quizMode === 'sound-to-char'}
                    onChange={(e) => setQuizMode(e.target.value as QuizMode)}
                  />
                  <span className={styles.radioCheck}></span>
                  <div className={styles.optionContent}>
                    <strong>Sound → Character</strong>
                    <span>Listen to sound, pick character</span>
                  </div>
                </label>

                <label className={styles.radioOption}>
                  <input
                    type="radio"
                    name="quizMode"
                    value="char-to-sound"
                    checked={quizMode === 'char-to-sound'}
                    onChange={(e) => setQuizMode(e.target.value as QuizMode)}
                  />
                  <span className={styles.radioCheck}></span>
                  <div className={styles.optionContent}>
                    <strong>Character → Sound</strong>
                    <span>See character, pick sound</span>
                  </div>
                </label>

                <label className={styles.radioOption}>
                  <input
                    type="radio"
                    name="quizMode"
                    value="romaji-to-char"
                    checked={quizMode === 'romaji-to-char'}
                    onChange={(e) => setQuizMode(e.target.value as QuizMode)}
                  />
                  <span className={styles.radioCheck}></span>
                  <div className={styles.optionContent}>
                    <strong>Romaji → Character</strong>
                    <span>Read romaji, pick character</span>
                  </div>
                </label>

                <label className={styles.radioOption}>
                  <input
                    type="radio"
                    name="quizMode"
                    value="char-to-romaji"
                    checked={quizMode === 'char-to-romaji'}
                    onChange={(e) => setQuizMode(e.target.value as QuizMode)}
                  />
                  <span className={styles.radioCheck}></span>
                  <div className={styles.optionContent}>
                    <strong>Character → Romaji</strong>
                    <span>See character, type romaji</span>
                  </div>
                </label>
              </div>
            </div>

            {/* Character Selection */}
            <div className={styles.setupSection}>
              <h3 className={styles.sectionTitle}>
                <span className={styles.sectionIcon}>📝</span>
                Character Rows
              </h3>
              <div className={styles.checkboxGroup}>
                {availableRows.map(row => (
                  <label key={row} className={styles.checkboxOption}>
                    <input
                      type="checkbox"
                      checked={selectedRows.includes(row)}
                      onChange={(e) => {
                        if (e.target.checked) {
                          setSelectedRows([...selectedRows, row]);
                        } else {
                          setSelectedRows(selectedRows.filter(r => r !== row));
                        }
                      }}
                    />
                    <span className={styles.checkboxCheck}></span>
                    <span className={styles.rowLabel}>{row}</span>
                  </label>
                ))}
              </div>
            </div>

            {/* Question Count */}
            <div className={styles.setupSection}>
              <h3 className={styles.sectionTitle}>
                <span className={styles.sectionIcon}>🔢</span>
                Number of Questions
              </h3>
              <div className={styles.dropdown}>
                <select
                  value={questionCount}
                  onChange={(e) => setQuestionCount(Number(e.target.value))}
                  className={styles.select}
                >
                  <option value={5}>5 Questions</option>
                  <option value={10}>10 Questions</option>
                  <option value={15}>15 Questions</option>
                  <option value={20}>20 Questions</option>
                  <option value={30}>30 Questions</option>
                </select>
              </div>
            </div>
          </div>

          <div className={styles.setupActions}>
            <button 
              className={styles.startBtn}
              onClick={startQuiz}
              disabled={selectedRows.length === 0}
            >
              <span className={styles.btnIcon}>🚀</span>
              Start Practice
            </button>
          </div>
        </div>
      </div>
    );
  }

  // Active Quiz Screen
  if (quizState === 'active' && questions.length > 0) {
    const question = questions[currentQuestion];
    return (
      <div className={styles.practiceMode}>
        <div className={styles.quizContainer}>
          {/* Quiz Header */}
          <div className={styles.quizHeader}>
            <div className={styles.progressSection}>
              <div className={styles.questionCounter}>
                Question {currentQuestion + 1} of {questions.length}
              </div>
              <div className={styles.progressBar}>
                <div 
                  className={styles.progressFill}
                  style={{ width: `${((currentQuestion + 1) / questions.length) * 100}%` }}
                ></div>
              </div>
              <div className={styles.scoreDisplay}>
                Score: {score}/{questions.length}
              </div>
            </div>
            
            <div className={styles.timer}>
              <div className={`${styles.timerCircle} ${timeLeft <= 5 ? styles.urgent : ''}`}>
                <span className={styles.timeLeft}>{timeLeft}</span>
              </div>
            </div>
          </div>

          {/* Question */}
          <div className={styles.questionSection}>
            <h2 className={styles.questionText}>{question.question}</h2>
            
            {(quizMode === 'sound-to-char' || quizMode === 'char-to-sound') && (
              <div className={styles.audioSection}>
                <button className={styles.playAudioBtn}>
                  <svg viewBox="0 0 24 24" fill="currentColor">
                    <path d="M8 5v14l11-7z"/>
                  </svg>
                  Play Sound
                </button>
              </div>
            )}

            {(quizMode === 'char-to-sound' || quizMode === 'romaji-to-char') && (
              <div className={styles.displayChar}>
                {quizMode === 'char-to-sound' ? question.correct.character : question.correct.romaji}
              </div>
            )}
          </div>

          {/* Answer Options */}
          <div className={styles.optionsGrid}>
            {question.options.map((option, index) => (
              <button
                key={index}
                className={`${styles.optionBtn} ${
                  selectedAnswer === (quizMode.includes('char') && quizMode.includes('romaji') ? option.romaji : option.character)
                    ? styles.selected : ''
                } ${showAnswer ? (
                  (quizMode.includes('char') && quizMode.includes('romaji') ? option.romaji : option.character) === 
                  (quizMode.includes('char') && quizMode.includes('romaji') ? question.correct.romaji : question.correct.character)
                    ? styles.correct : styles.incorrect
                ) : ''}`}
                onClick={() => !showAnswer && handleAnswer(
                  quizMode.includes('char') && quizMode.includes('romaji') ? option.romaji : option.character
                )}
                disabled={showAnswer}
              >
                <div className={styles.optionContent}>
                  <span className={styles.optionChar}>
                    {quizMode === 'sound-to-char' || quizMode === 'romaji-to-char' 
                      ? option.character 
                      : option.romaji
                    }
                  </span>
                </div>
              </button>
            ))}
          </div>
        </div>
      </div>
    );
  }

  // Results Screen
  if (quizState === 'results') {
    const percentage = Math.round((score / questions.length) * 100);
    return (
      <div className={styles.practiceMode}>
        <div className={styles.resultsContainer}>
          <div className={styles.resultsHeader}>
            <div className={styles.resultsIcon}>
              {percentage >= 80 ? '🎉' : percentage >= 60 ? '👍' : '📚'}
            </div>
            <h2 className={styles.resultsTitle}>Quiz Complete!</h2>
          </div>

          <div className={styles.scoreCard}>
            <div className={styles.finalScore}>
              <span className={styles.scoreNumber}>{score}</span>
              <span className={styles.scoreDivider}>/</span>
              <span className={styles.totalQuestions}>{questions.length}</span>
            </div>
            <div className={styles.percentage}>{percentage}%</div>
            
            <div className={styles.performanceMessage}>
              {percentage >= 90 && "Excellent! You're mastering this!"}
              {percentage >= 80 && percentage < 90 && "Great job! Keep practicing!"}
              {percentage >= 60 && percentage < 80 && "Good work! Room for improvement."}
              {percentage < 60 && "Keep practicing! You'll get there!"}
            </div>
          </div>

          <div className={styles.resultsActions}>
            <button className={styles.retryBtn} onClick={resetQuiz}>
              <span className={styles.btnIcon}>🔄</span>
              Practice Again
            </button>
            <button className={styles.newQuizBtn} onClick={resetQuiz}>
              <span className={styles.btnIcon}>⚙️</span>
              New Settings
            </button>
            <button className={styles.exitBtn} onClick={onExit}>
              <span className={styles.btnIcon}>🏠</span>
              Back to Charts
            </button>
          </div>
        </div>
      </div>
    );
  }

  return null;
};

export default PracticeMode;
