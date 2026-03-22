package dhids

// AdversaryID identifies a Daggerheart adversary.
type AdversaryID string

func (id AdversaryID) String() string { return string(id) }

// EnvironmentEntityID identifies an instantiated Daggerheart environment.
type EnvironmentEntityID string

func (id EnvironmentEntityID) String() string { return string(id) }

// CountdownID identifies a Daggerheart countdown.
type CountdownID string

func (id CountdownID) String() string { return string(id) }
