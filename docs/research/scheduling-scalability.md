EvoFarm

Learning-Augmented Optimization Infrastructure for Large-Scale Scheduling

EvoFarm is a distributed optimization platform exploring how evolutionary computation, constraint programming, mathematical optimization, heuristics, and machine learning can work together to solve increasingly large and dynamic scheduling problems.

The central idea is not to replace exact optimization with machine learning.

Instead:

Use the right optimization technique for the right part of the problem, and use learning to make those techniques more effective.

⸻

Why EvoFarm?

Scheduling systems become difficult for reasons that go beyond the raw number of variables.

As scheduling problems grow, they may involve:

* thousands of jobs
* hundreds of machines/resources
* complex precedence relationships
* resource conflicts
* sequence-dependent setup times
* cumulative constraints
* changing priorities
* machine failures
* new job arrivals
* uncertain processing times
* strict response-time requirements

No single optimization technique is ideal across all of these scenarios.

Exact solvers such as OR-Tools CP-SAT are extremely powerful for hard-constrained optimization, but their practical scalability depends heavily on model structure, constraint density, numerical representation, memory consumption, and search complexity.

At the other extreme, machine-learning policies can make extremely fast decisions after training, but they do not naturally provide the same constraint guarantees or optimality guarantees as mathematical optimization.

EvoFarm explores the space between these approaches.

⸻

Core Vision

EvoFarm aims to become a:

Learning-Augmented Optimization Platform

rather than a pure evolutionary algorithm framework or a pure reinforcement-learning scheduler.

The long-term architecture combines:

                    Scheduling Problem
                           │
                           ▼
                 ┌────────────────────┐
                 │ Problem Analyzer    │
                 └─────────┬──────────┘
                           │
                           ▼
                 ┌────────────────────┐
                 │ Optimization Router│
                 └─────────┬──────────┘
                           │
             ┌─────────────┼─────────────┐
             │             │             │
             ▼             ▼             ▼
          CP-SAT      Column Generation  Heuristics
             │             │             │
             │             │             │
             └─────────────┼─────────────┘
                           ▼
                  Candidate Solutions
                           │
                           ▼
                 ┌────────────────────┐
                 │ EvoFarm Learning   │
                 │ / Evolution / RL   │
                 └─────────┬──────────┘
                           │
                    guide / select /
                    repair / improve
                           │
                           ▼
                 ┌────────────────────┐
                 │ Constraint Checker │
                 └─────────┬──────────┘
                           │
                           ▼
                    Final Schedule

The objective is not to make every component solve the entire problem.

Each component should perform the task it is best suited for.

⸻

Optimization Philosophy

EvoFarm follows five principles.

1. Exact optimization for guarantees

Use CP-SAT or MILP when:

* hard constraints dominate
* problem size is manageable
* feasibility is critical
* optimality or optimality gaps matter

2. Decomposition for structural scalability

When a problem has exploitable structure, divide it into smaller subproblems.

Potential approaches include:

* Dantzig-Wolfe decomposition
* column generation
* Benders decomposition
* parallel pricing
* hierarchical optimization

3. Large Neighborhood Search for improvement

Instead of repeatedly solving an entire scheduling problem, freeze most of an existing solution and re-optimize selected regions.

Current Schedule
       │
       ▼
Select Neighborhood
       │
       ▼
Destroy / Relax
       │
       ▼
CP-SAT / MILP
       │
       ▼
Repair / Improve
       │
       ▼
Better Schedule

This creates a natural bridge between exact optimization and heuristic search.

4. Machine learning for repeated decisions

Learning is most useful where the same type of decision occurs repeatedly.

Examples:

* job dispatching
* machine selection
* variable selection
* branching decisions
* neighborhood selection
* heuristic selection
* warm-start generation
* rescheduling decisions

5. Validation remains deterministic

A learned policy should not be trusted as the final authority when hard constraints matter.

A preferred architecture is:

ML / RL
   │
   ▼
Candidate Decision
   │
   ▼
Constraint Validation
   │
   ▼
Repair / Optimization
   │
   ▼
Deployable Schedule

⸻

Why Not Just CP-SAT?

OR-Tools CP-SAT is one of the most capable open-source constraint optimization tools available and should remain an important component of EvoFarm.

However, CP-SAT does not have a simple universal scalability threshold such as “one million variables.”

Practical performance depends on:

* variable domains
* constraint structure
* constraint density
* interval variables
* NoOverlap
* cumulative constraints
* precedence structure
* symmetry
* coefficient magnitudes
* presolve behavior
* search complexity
* memory availability
* number of workers
* objective structure

Two models with the same number of variables can have dramatically different solving times.

Therefore, EvoFarm treats problem structure, rather than raw variable count, as the primary indicator of scalability.

⸻

Production Failure Modes

Large optimization models can introduce operational problems beyond solution quality.

Examples include:

Model construction limits

Very large models can encounter numerical or internal representation limits.

For example:

Invalid model:
The sum of all variable domains do not fit on an int64_t

These errors are not evidence that CP-SAT has a universal variable-count limit.

They demonstrate that model representation and numerical bounds can become constraints themselves.

Long solver shutdown

Very large or pathological models can also exhibit significant termination delays relative to the requested wall-time limit.

This is especially important for production systems with strict latency requirements.

Therefore, EvoFarm should treat optimization workers as independently managed processes.

                    Scheduler API
                         │
                         ▼
                 Scheduler Controller
                         │
          ┌──────────────┼──────────────┐
          ▼              ▼              ▼
      CP-SAT Worker   Heuristic       ML Worker
          │              │              │
          └──────────────┼──────────────┘
                         ▼
                    Candidate Pool

The controller can enforce an external process/container timeout rather than relying exclusively on an optimizer’s internal time limit.

⸻

The Dense Interference Problem

Scheduling difficulty is often driven more by interaction density than by variable count.

Consider:

Job A ───────┐
             │
Job B ───────┼── Resource R1
             │
Job C ───────┘
Job A ───────┐
             │
Job D ───────┼── Resource R2
             │
Job E ───────┘

As resource conflicts, precedence relationships, synchronization requirements, and setup dependencies increase, the search space becomes increasingly interconnected.

Therefore:

A 10,000-variable sparse problem may be easier than a 2,000-variable highly interconnected problem.

This is one of the reasons EvoFarm focuses on structural analysis and hybrid optimization rather than simply increasing solver resources.

⸻

Scheduling Scale Dimensions

Problem scale should not be measured only by variable count.

EvoFarm considers four dimensions.

1. Model Scale

* jobs
* machines
* operations
* time periods
* scenarios
* variables
* constraints

2. Interaction Density

* precedence edges
* resource conflicts
* setup dependencies
* synchronization
* cumulative resources

3. Dynamic Complexity

* job arrival rate
* machine failures
* priority changes
* processing-time uncertainty
* environmental changes

4. Decision Latency

Different applications require different response times.

Offline planning       → hours
Operational planning   → minutes
Dynamic rescheduling   → seconds
Real-time control      → milliseconds

A model that takes 10 minutes may be excellent for overnight planning and completely useless for a 500 ms decision.

⸻

Hybrid Optimization Architecture

The target architecture is a solver portfolio.

Instead of:

Problem → CP-SAT

EvoFarm evolves toward:

Problem
  │
  ▼
Problem Characterization
  │
  ▼
Optimization Router
  │
  ├── CP-SAT
  │
  ├── MILP
  │
  ├── Column Generation
  │
  ├── Large Neighborhood Search
  │
  ├── Evolutionary Search
  │
  └── ML / RL Policy
          │
          ▼
    Candidate Solutions
          │
          ▼
    Validation / Repair
          │
          ▼
      Final Solution

The router can eventually learn which strategy works best for a particular problem class.

⸻

Machine Learning’s Role

Machine learning should not initially replace the optimizer.

Instead, ML can learn decisions around the optimizer.

Potential applications include:

Warm starts

Problem
   │
   ▼
Neural Model
   │
   ▼
Initial Schedule
   │
   ▼
CP-SAT
   │
   ▼
Improved Schedule

Dispatching

Available Jobs
      │
      ▼
RL Policy
      │
      ▼
Select Next Job
      │
      ▼
Constraint Validation

Neighborhood selection

Current Schedule
       │
       ▼
ML Policy
       │
       ▼
Select Jobs / Region
       │
       ▼
Destroy + Repair
       │
       ▼
CP-SAT

Heuristic selection

The model can learn which optimization strategy to invoke:

                Problem
                   │
                   ▼
              ML Router
                   │
       ┌───────────┼───────────┐
       ▼           ▼           ▼
     CP-SAT       LNS      Evolution

This is potentially more robust than asking a neural network to directly produce an entire schedule.

⸻

Reinforcement Learning

RL is particularly interesting for repeated and dynamic scheduling decisions.

Potential applications:

* dispatching
* reactive scheduling
* machine assignment
* buffer allocation
* neighborhood selection
* heuristic selection
* resource allocation

For example:

                    Event
                      │
                      ▼
              Current Schedule
                      │
                      ▼
                  RL Policy
                      │
          ┌───────────┼───────────┐
          ▼           ▼           ▼
      Reschedule   Keep Job    Change Machine
          │           │           │
          └───────────┼───────────┘
                      ▼
               Constraint Check
                      │
                      ▼
                Repair / Optimize

However, EvoFarm does not assume that pure RL is automatically superior to optimization.

RL introduces its own challenges:

* reward design
* training cost
* generalization
* distribution shift
* unseen problem sizes
* constraint violations
* reproducibility
* explainability

Therefore, learned policies should generally operate within a constrained optimization pipeline.

⸻

Decomposition and Column Generation

For very large structured problems, decomposition can be more important than simply adding more compute.

Dantzig-Wolfe decomposition transforms a large optimization problem into:

                 Master Problem
                       │
                       │ dual prices
                       ▼
                Pricing Problems
                 │      │      │
                 ▼      ▼      ▼
               Worker Worker Worker
                 │      │      │
                 └──────┼──────┘
                        │
                        ▼
                    New Columns
                        │
                        ▼
                 Master Problem

This creates natural opportunities for distributed computation.

EvoFarm workers can potentially act as parallel pricing solvers.

⸻

Very Large Neighborhood Search

Large Neighborhood Search is another important bridge between optimization and learning.

Instead of exploring tiny local changes:

Swap Job A and Job B

the system can optimize an entire region:

Freeze 90%
Optimize 10%

For example:

J1 J2 J3 J4 J5 J6 J7 J8 J9 J10
│  │  │  │  │  │  │  │  │  │
│  │  └──────────────┘  │  │
│  │       relax        │  │
│  └────────────────────┴──┘
            │
            ▼
          CP-SAT

ML can eventually learn:

Which 10% should be optimized next?

This is one of the most promising hybrid directions for EvoFarm.

⸻

Decision Matrix

Scheduling Regime	Recommended Approach
Small + highly constrained	CP-SAT / MILP
Medium + complex	CP-SAT + LNS
Large + structured/decomposable	Column Generation / Dantzig-Wolfe
Large + repeated	Decomposition + heuristics/metaheuristics
Dynamic scheduling	Predictive schedule + event-driven repair
Dynamic + repeated	ML/RL-guided dispatching or LNS
Very low latency	Learned policy + constraint checker + repair
Safety-critical	Validated optimization as final authority

The architecture should therefore be selected based on:

Problem structure
       +
Solution quality requirement
       +
Constraint strictness
       +
Decision latency
       +
Frequency of repeated decisions

rather than variable count alone.

⸻

EvoFarm Roadmap

Phase 1 — Distributed Neuroevolution

Status: Complete

Current capabilities:

* distributed evolutionary computation
* XOR and simple optimization problems
* containerized workers
* polyglot architecture
* distributed execution

Demonstrates:

* distributed systems
* evolutionary algorithms
* containerization
* worker orchestration

⸻

Phase 2 — Constraint Optimization

Goal: Introduce exact optimization as a complementary capability.

Planned API

POST /solve/constraint

Initial use cases

* nurse rostering
* school timetabling
* workforce scheduling
* resource allocation

Architecture:

Client
  │
  ▼
EvoFarm API
  │
  ▼
Constraint Worker
  │
  ▼
OR-Tools CP-SAT
  │
  ▼
Validated Schedule

The goal is not to replace neuroevolution.

It is to demonstrate that EvoFarm can select the appropriate optimization paradigm.

⸻

Phase 3 — LNS + ML-Guided Optimization

Goal: Build the first meaningful hybrid optimization system.

Instead of immediately training a neural network to generate complete schedules, begin with a safer architecture.

Step 1 — ML-guided dispatching

Learn:

Which job should execute next?

Step 2 — ML-guided neighborhood selection

Learn:

Which jobs should be reoptimized?

Step 3 — Neural warm starts

Generate:

Problem → Neural Policy → Initial Schedule → CP-SAT

Step 4 — Benchmark

Compare:

CP-SAT
CP-SAT + LNS
CP-SAT + ML warm start
CP-SAT + ML-guided LNS

Measure:

* feasibility
* objective value
* optimality gap
* time to first feasible solution
* time to best solution
* memory
* scalability
* robustness

This phase provides a measurable research contribution rather than simply adding an RL model.

⸻

Phase 4 — Column Generation

Goal: Support large decomposable scheduling problems.

Architecture:

                  Master Problem
                       │
              ┌────────┼────────┐
              ▼        ▼        ▼
           Worker   Worker   Worker
              │        │        │
              ▼        ▼        ▼
           Pricing  Pricing  Pricing
              │        │        │
              └────────┼────────┘
                       ▼
                  New Columns

Potential use cases:

* vehicle routing
* crew scheduling
* large-scale assignment
* resource allocation
* unit commitment
* decomposition-based scheduling

EvoFarm workers can become distributed pricing engines.

⸻

Phase 5 — Dynamic Rescheduling

Goal: Build an event-driven predictive-reactive scheduling system.

Predictive layer

Generate an initial schedule.

Orders
  │
  ▼
Optimizer
  │
  ▼
Baseline Schedule

Reactive layer

Monitor:

Machine failure
New order
Cancelled order
Priority change
Resource unavailable
Processing delay

Then:

Event
  │
  ▼
Impact Analysis
  │
  ▼
RL / ML Policy
  │
  ▼
Select Rescheduling Strategy
  │
  ▼
LNS / CP-SAT / Heuristic
  │
  ▼
Validated Schedule

⸻

Phase 6 — Intelligent Optimization Router

Long-term goal

EvoFarm should eventually learn:

Which optimization strategy should be used for this problem?

For example:

                  Scheduling Problem
                         │
                         ▼
                Problem Characterizer
                         │
                         ▼
                Intelligent Router
                         │
       ┌─────────────────┼─────────────────┐
       ▼                 ▼                 ▼
    CP-SAT             LNS            Column Generation
       │                 │                 │
       └─────────────────┼─────────────────┘
                         ▼
                  Candidate Solutions
                         │
                         ▼
                  Improvement Layer
                         │
                         ▼
                    Validation

The router itself could eventually use machine learning.

This creates a second-order optimization problem:

Learn how to choose the optimizer.

⸻

Research Questions

EvoFarm can be used to investigate several research questions.

RQ1 — Can ML improve CP-SAT warm starts?

Compare:

Random / heuristic initialization
vs.
ML-generated initialization

⸻

RQ2 — Can ML select useful LNS neighborhoods?

Compare:

Random neighborhoods
vs.
Hand-designed neighborhoods
vs.
ML-selected neighborhoods

⸻

RQ3 — When does decomposition outperform monolithic optimization?

Measure:

* problem size
* constraint density
* decomposition overhead
* parallelism
* optimality gap
* runtime

⸻

RQ4 — Can learned policies generalize across problem sizes?

Train on:

100 jobs

Test on:

200
500
1,000
5,000 jobs

This is critical because generalization is one of the major challenges of learning-based combinatorial optimization.

⸻

RQ5 — Can EvoFarm dynamically select optimization strategies?

Given a new problem:

Should we use:
CP-SAT?
LNS?
Column Generation?
Evolution?
RL?
Hybrid?

Can a learned router make that decision effectively?

⸻

Evaluation Framework

Every new optimization strategy should be evaluated against strong baselines.

Solution Quality

Measure:

* objective value
* constraint violations
* optimality gap
* number of feasible solutions

Computational Performance

Measure:

* time to first feasible solution
* time to best solution
* total runtime
* CPU utilization
* memory consumption

Scalability

Test across:

100
500
1,000
5,000
10,000
50,000+

where the problem formulation makes those scales meaningful.

Do not interpret these numbers as universal CP-SAT limits.

The objective is to characterize the behavior of different architectures.

Robustness

Test:

* random disruptions
* machine failures
* new jobs
* changing priorities
* incomplete information
* unusual problem structures

⸻

Benchmarking Philosophy

EvoFarm should never claim:

“Our ML approach is better.”

without specifying:

Better at what?

Every experiment should define:

Objective
Constraints
Problem distribution
Latency requirement
Baseline
Hardware
Random seeds
Stopping criteria

For example:

Baseline:
CP-SAT, 300 seconds
Hybrid:
ML-guided LNS, 30 seconds
Compare:
- feasibility rate
- objective gap
- runtime
- robustness

This makes experimental results reproducible and meaningful.

⸻

Research Positioning

EvoFarm sits at the intersection of:

Operations Research
        │
        ├── Constraint Programming
        ├── Mathematical Optimization
        ├── Column Generation
        └── Local Search
                │
                ▼
        Hybrid Optimization
                │
        ┌───────┴───────┐
        ▼               ▼
 Machine Learning   Evolutionary
        │            Computation
        └───────┬───────┘
                ▼
        Intelligent Scheduling

The long-term research direction is:

Learning-Augmented Combinatorial Optimization

⸻

Key Takeaways

Exact solvers remain essential

CP-SAT and MILP are not obsolete.

They provide:

* strong constraint handling
* high-quality solutions
* optimality guarantees in appropriate settings
* deterministic validation

⸻

But exact optimization is not universally sufficient

Large, highly interconnected, dynamic, or latency-sensitive problems can require additional techniques.

⸻

Decomposition is a fundamental scalability mechanism

For structured problems, breaking a large optimization problem into smaller subproblems can provide much greater scalability than simply increasing solver resources.

⸻

LNS provides an important middle ground

It allows an existing high-quality schedule to be improved without rebuilding the entire solution from scratch.

⸻

Machine learning should guide optimization

Rather than immediately replacing optimization:

ML
 ↓
Guide
 ↓
Optimize
 ↓
Validate

This is the core hybrid philosophy of EvoFarm.

⸻

Reinforcement learning is particularly promising for dynamic decisions

RL can learn policies for:

* dispatching
* rescheduling
* neighborhood selection
* resource allocation
* heuristic selection

But RL should not be assumed to dominate exact optimization in every environment.

⸻

Interview Narrative

When discussing EvoFarm in an interview, the core argument is:

“I don’t believe scaling scheduling means simply throwing more compute at CP-SAT, nor do I think the answer is to replace optimization with an RL model. Different scheduling regimes require different optimization strategies. Exact solvers are excellent when hard constraints and optimality matter, decomposition helps when the problem has exploitable structure, LNS and heuristics provide scalable improvement, and machine learning can learn repeated decisions such as dispatching, neighborhood selection, and warm starts.

EvoFarm is exploring how these techniques can be combined into a learning-augmented optimization platform where ML proposes or selects decisions, while deterministic optimization and constraint validation provide reliability.”

This is the central architectural thesis of the project.

⸻

References

Neural Combinatorial Optimization

Chung, K. T., Lee, C. K. M., & Tsang, Y. P. (2025).

Neural combinatorial optimization with reinforcement learning in industrial engineering: a survey.

Artificial Intelligence Review, 58(5).

DOI: 10.1007/s10462-024-11045-1

⸻

Dynamic Scheduling with Deep Reinforcement Learning

Deep reinforcement learning for event-driven predictive–reactive multi-objective scheduling in dynamic flexible job shop.

(2025)

ScienceDirect.

⸻

Hierarchical Reinforcement Learning

Buffer-adaptive hierarchical deep reinforcement learning for multi-line Hybrid Flow Shop Scheduling with Limited Buffers.

(2026)

Engineering Applications of Artificial Intelligence.

⸻

Large-Scale Column Generation

Su Zhaogang, Tang Yuyang, Chen Shengjie, Chen Liang, Deng Jiayi. (2026).

An Efficient Two-Stage Column Generation Algorithm for Solving Large-Scale Unit Commitment.

Mathematica Numerica Sinica, 48(1), 181–210.

DOI: 10.12286/jssx.j2025-1330

⸻

Stochastic Decomposition

Rathi, T., Riley, B. P., Flores-Quiroz, A., & Zhang, Q. (2026).

Column generation for multistage stochastic mixed-integer nonlinear programs with discrete state variables.

Journal of Global Optimization, 94(1), 95.

DOI: 10.1007/s10898-025-01480-x

⸻

Resource-Constrained Project Scheduling

Chen, L., Zhang, J., Chen, Z., & Demeulemeester, E. (2025).

Resource-constrained project scheduling problem with hybrid energy and dynamic energy prices.

International Journal of Production Research, 63(23), 9155–9180.

DOI: 10.1080/00207543.2025.2535516

⸻

Large Neighborhood / Very Large-Scale Search

Local search intensified: Very large-scale variable neighborhood search for the multi-resource generalized assignment problem.

This work provides foundational background for VLSN-based improvement strategies.

⸻

CP-SAT Operational Behavior

CP-SAT solver takes way too long to shut down or does not stop at all despite given time limit.

OR-Tools GitHub Issue #4882.

This issue is treated as evidence of a specific operational failure mode rather than evidence of a universal CP-SAT time-limit guarantee failure.

⸻

Project Status

Phase	Component	Status
1	Distributed Neuroevolution	✅ Complete
2	CP-SAT Integration	🔜 Planned
3	LNS + ML Guidance	🔜 Planned
4	Column Generation	🔜 Planned
5	Dynamic Rescheduling	🔜 Planned
6	Intelligent Optimization Router	🔭 Research

⸻

Long-Term Vision

EvoFarm is ultimately exploring a scheduling architecture where:

                  ┌──────────────────────┐
                  │ Scheduling Environment│
                  └──────────┬───────────┘
                             │
                             ▼
                   Problem Characterization
                             │
                             ▼
                    Intelligent Optimizer
                             │
          ┌──────────────────┼──────────────────┐
          │                  │                  │
          ▼                  ▼                  ▼
       CP-SAT          Column Generation       LNS
          │                  │                  │
          └──────────────────┼──────────────────┘
                             ▼
                       Candidate Pool
                             │
                             ▼
                    Evolution / ML / RL
                             │
                             ▼
                     Repair / Improve
                             │
                             ▼
                    Constraint Validation
                             │
                             ▼
                       Final Schedule
                             │
                             ▼
                         Execution
                             │
                             ▼
                           Events
                             │
                             └──────────────┐
                                            │
                                            ▼
                                      Rescheduling

The goal is not to build another scheduler that assumes one algorithm solves everything.

The goal is to build infrastructure that can learn which optimization strategy to use, combine multiple strategies, and continuously improve scheduling decisions under real-world constraints.

⸻

License

See the repository license for details.

⸻

EvoFarm — Exploring the future of learning-augmented optimization.